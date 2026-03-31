---
name: harvest-annotate
description: Annotate Harvest time entries with Jira task numbers based on git commits and Claude conversation history. Use at end of day to fill in missing Jira task references.
disable-model-invocation: true
allowed-tools: Bash(*bin/harvest-cli *), Bash(*bin/conversation-extract *), Bash(git -C * log *), Bash(git -C * show *), Bash(git -C * diff *), Bash(git log *), Bash(git show *), Bash(git diff *), Read, Grep, Glob, Edit, AskUserQuestion, mcp__atlassian__*list*, mcp__atlassian__*get*, mcp__atlassian__*search*, mcp__atlassian__*List*, mcp__atlassian__*Get*, mcp__atlassian__*Search*
argument-hint: "[today | yesterday | past working week | YYYY-MM-DD | --from DATE --to DATE]"
---

# Harvest Time Entry Annotator

You are annotating Harvest time entries with Jira task numbers.

**CLI path:** `${CLAUDE_SKILL_DIR}/../../bin/harvest-cli`
**Project-repo mapping:** User-specific at `${CLAUDE_PLUGIN_DATA}/repos.json`

## Phase 0: Determine date range

The user's arguments: `$ARGUMENTS`
Today's date is available in the system context as `currentDate`. Use it directly — do not shell out for the date.

Interpret the arguments to determine `--from` and `--to` dates (YYYY-MM-DD format):
- No arguments: use today's date.
- `today` / `yesterday`: self-explanatory
- `past working week` / `last week`: Monday through Friday of the most recent working week
- `this week`: Monday through today
- A specific date like `2026-03-28`: use that single date
- `--from YYYY-MM-DD --to YYYY-MM-DD`: use as-is

## Phase 1: Fetch Harvest entries

Run the CLI to fetch entries for the determined date range:
```bash
HARVEST_DATA_DIR=${CLAUDE_PLUGIN_DATA} ${CLAUDE_SKILL_DIR}/../../bin/harvest-cli fetch --from <FROM> --to <TO>
```

**If it fails with "no auth config found" or an auth error:**
1. Tell the user to create a personal access token at https://id.getharvest.com/developers
2. Instruct them to fill in `${CLAUDE_PLUGIN_DATA}/auth.json`:
   > Please edit `${CLAUDE_PLUGIN_DATA}/auth.json` with your credentials:
   > ```json
   > {"access_token": "YOUR_TOKEN", "account_id": "YOUR_ACCOUNT_ID"}
   > ```
   > Run `! vi ${CLAUDE_PLUGIN_DATA}/auth.json` in the prompt to edit it.
3. Stop and wait for the user to confirm they've done it, then re-run the fetch command

**If it fails with an HTTP error:** show the error and stop.

## Phase 2: Identify unlinked entries

The fetch command already returns only the current user's entries.

For each entry:

1. **Extract Jira project key(s)** from the entry's `project.code` field:
   - If `code` is null or empty: the project has no Jira association — skip it
   - If `code` contains `|` (e.g., `"VMOB|MCVIR"`): split on `|` to get multiple possible Jira project keys
   - Otherwise use the code directly as a single-element list of project keys
   - If ANY of the resulting project keys are in the `_no_jira` array in `repos.json`: skip the entry (no annotation needed)
2. **Check if notes already contain a task number** matching `KEY-\d+` for ANY of the project keys (e.g., `VMOB-123` or `MCVIR-456`)
3. Entries WITHOUT a matching task number are "unlinked"

If all entries already have task numbers (or are skipped), report the status and stop.

Show a brief summary:
- Total entries in date range (for this user)
- Already linked (with their task numbers)
- Skipped (no Jira project or in `_no_jira`)
- Unlinked (need annotation)

## Phase 3: Gather work context for each unlinked entry

Read the project-repo mapping from `${CLAUDE_PLUGIN_DATA}/repos.json`. If the file doesn't exist, create it with `{}`.

Check the `_no_repo` array in `repos.json`. Skip git/conversation context gathering for any project key listed there — proceed directly to Phase 4 for those entries.

For each remaining unlinked entry, use the first Jira project key from its `project.code`, then:

**If the project key is NOT in the mapping:**
- Ask the user which repo directories are associated with this Jira project. The user may also say there is no repo — if so, add the key to the `_no_repo` array.
- Update `${CLAUDE_PLUGIN_DATA}/repos.json` with the Edit tool
- Note: one key can map to multiple repos, and one repo can appear under multiple keys

**Gather context in parallel.** Issue all git and conversation-extract calls as separate, parallel tool calls in a single response (do not chain commands with `&&` or `;`):

Git commits:
```bash
git -C <repo_path> log --since="<FROM> 00:00" --until="<TO> 23:59" --format="%h %s (%an)" --all 2>/dev/null
```

Claude conversation history:
```bash
${CLAUDE_SKILL_DIR}/../../bin/conversation-extract <repo_path> <FROM> <TO>
```
This returns AI-generated summaries of conversations from the date range.

**Build a work summary** combining git commits + conversation topics.

## Phase 4: Find matching Jira tasks

Check the `_external_jira` array in `repos.json`. Skip MCP lookup for any project key listed there.

For each remaining unlinked entry, you MUST attempt a Jira search before presenting options to the user. Issue all searches as parallel tool calls in a single response:

1. Call `mcp__atlassian__searchJiraIssuesUsingJql` for each project key with:
   ```
   JQL: project = XXX AND (status != Done OR updated >= -7d) ORDER BY updated DESC
   ```
2. Get details for top candidates with `mcp__atlassian__getJiraIssue`
3. Match the work summary against issue summaries and present the best matches as options in Phase 5

**If the `mcp__atlassian__*` tools are not listed in your available tools**, tell the user: "Atlassian MCP server is not available. Check your MCP configuration and authentication." Then **stop and wait** for the user to either fix it and restart, or explicitly instruct you to proceed without Jira suggestions.

**If a tool call fails** (returns an error), show the error to the user. Ask if this project is on an external Jira, and if so, add the project key to the `_external_jira` array in `repos.json` so future runs skip MCP lookup automatically.

## Phase 5: Interactive annotation

For each unlinked entry, present context then ask the user:

```
--- Harvest Entry ---
Project: <project name> (code: <code>)
Hours:   X.X
Notes:   "current notes"
Date:    YYYY-MM-DD

--- Work Done ---
- Summary of commits and conversation topics

--- Suggested Jira Tasks ---
```

Use AskUserQuestion with top matching tasks as options. The user can also type a task key directly, skip, or say this project never needs Jira tasks — if so, add the project key to `_no_jira` in `repos.json` so future runs skip it automatically.

## Phase 6: Update Harvest

For confirmed matches, **prepend** the task key to existing notes:
```bash
HARVEST_DATA_DIR=${CLAUDE_PLUGIN_DATA} ${CLAUDE_SKILL_DIR}/../../bin/harvest-cli update-notes <entry_id> "XXX-123 existing notes"
```

## Phase 7: Summary

```
Results:
  Updated: X entries
    <project> X.Xh -> TASK-123
    ...
  Skipped: X entries
  No Jira project: X entries
```
