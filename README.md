# harvest-annotate-plugin

A Claude Code plugin that annotates [Harvest](https://www.getharvest.com/) time entries with Jira task numbers by correlating git commits and Claude conversation history with active Jira issues.

## What it does

At the end of a workday (or week), invoke the `harvest-annotate` skill to:

1. Fetch your Harvest time entries for a given date range
2. Identify entries missing Jira task references
3. Gather work context from git commit history and Claude conversation logs
4. Search Jira for matching issues via the Atlassian MCP
5. Interactively suggest and apply task annotations to your Harvest entries

## Prerequisites

- **Claude Code** with plugin support
- **Harvest API credentials** -- a personal access token and account ID from [Harvest](https://id.getharvest.com/developers)
- **Atlassian MCP** -- the [Atlassian Remote MCP Server](https://www.atlassian.com/platform/remote-mcp-server) must be configured in Claude Code for Jira issue lookup

## Installation

### 1. Install the Atlassian MCP server

Add the Atlassian remote MCP server to your global Claude Code settings (`~/.claude/settings.json`):

```json
{
  "mcpServers": {
    "atlassian": {
      "type": "url",
      "url": "https://mcp.atlassian.com/v1/sse"
    }
  }
}
```

You will be prompted to authenticate with your Atlassian account on first use.

### 2. Install the plugin via Claude Code marketplace

1. Run `/plugin` in Claude Code
2. Go to the **Marketplaces** tab
3. Select **Add Marketplace** and enter `tmikoss/harvest-annotate-plugin`
4. Follow the prompts to install

### 3. Configure Harvest credentials

On first session start, the plugin creates a data directory with placeholder config files. Edit the `auth.json` file in the plugin data directory and fill in your Harvest credentials:

```json
{
  "access_token": "YOUR_HARVEST_PERSONAL_ACCESS_TOKEN",
  "account_id": "YOUR_HARVEST_ACCOUNT_ID"
}
```

### 4. Project-to-repo mappings

The plugin maintains a `repos.json` file in the plugin data directory that maps Jira project keys to local repository paths. Claude manages this file automatically -- it will prompt you to add mappings for unknown projects during annotation.

## Usage

Invoke the skill in any Claude Code session:

```
/harvest-annotate today
/harvest-annotate yesterday
/harvest-annotate past working week
/harvest-annotate 2026-03-28
/harvest-annotate --from 2026-03-24 --to 2026-03-28
```

## Building from source

The plugin ships with pre-built binaries. To rebuild:

```sh
cd bin
make all
```

Requires Go 1.19+.
