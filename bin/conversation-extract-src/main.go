package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const summarizePrompt = `Below is a Claude Code conversation transcript. Summarize what was worked on in 2-3 brief bullet points. Focus on concrete tasks, features, or bugs. Output only the bullet points, nothing else.`

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: conversation-extract <repo_path> <from> <to>")
		fmt.Fprintln(os.Stderr, "  repo_path: absolute path to the repo")
		fmt.Fprintln(os.Stderr, "  from/to:   YYYY-MM-DD date range")
		os.Exit(1)
	}

	repoPath := os.Args[1]
	fromStr := os.Args[2]
	toStr := os.Args[3]

	from, err := time.ParseInLocation("2006-01-02", fromStr, time.Local)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid from date: %v\n", err)
		os.Exit(1)
	}
	to, err := time.ParseInLocation("2006-01-02", toStr, time.Local)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid to date: %v\n", err)
		os.Exit(1)
	}
	// Include the full "to" day
	to = to.Add(24*time.Hour - time.Second)

	projectDir := claudeProjectDir(repoPath)
	if projectDir == "" {
		os.Exit(0)
	}

	entries, err := os.ReadDir(projectDir)
	if err != nil {
		os.Exit(0)
	}

	type result struct {
		index   int
		summary string
	}

	var conversations []struct {
		name string
		text string
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		modTime := info.ModTime()
		if modTime.Before(from) || modTime.After(to) {
			continue
		}

		path := filepath.Join(projectDir, entry.Name())
		text := extractConversation(path)
		if text == "" {
			continue
		}
		conversations = append(conversations, struct {
			name string
			text string
		}{entry.Name(), text})
	}

	results := make(chan result, len(conversations))
	sem := make(chan struct{}, 10)
	for i, conv := range conversations {
		go func(idx int, name, text string) {
			sem <- struct{}{}
			defer func() { <-sem }()
			summary, err := summarize(text)
			if err != nil {
				fmt.Fprintf(os.Stderr, "summarize failed for %s: %v\n", name, err)
				results <- result{idx, ""}
				return
			}
			results <- result{idx, summary}
		}(i, conv.name, conv.text)
	}

	summaries := make([]string, len(conversations))
	for range conversations {
		r := <-results
		summaries[r.index] = r.summary
	}

	for _, s := range summaries {
		if s != "" {
			fmt.Println(s)
		}
	}
}

func claudeProjectDir(repoPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot determine home directory: %v\n", err)
		return ""
	}
	encoded := strings.TrimPrefix(repoPath, "/")
	encoded = strings.ReplaceAll(encoded, "/", "-")
	return filepath.Join(home, ".claude", "projects", "-"+encoded)
}

func summarize(conversationText string) (string, error) {
	cmd := exec.Command("claude", "-p", summarizePrompt, "--model", "haiku")
	cmd.Stdin = strings.NewReader(conversationText)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%w: %s", err, exitErr.Stderr)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type message struct {
	Type    string         `json:"type"`
	Message messageContent `json:"message"`
}

type messageContent struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func extractConversation(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var msg message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}

		var role string
		switch msg.Type {
		case "user":
			role = "USER"
		case "assistant":
			role = "ASSISTANT"
		default:
			continue
		}

		text := extractText(msg.Message.Content)
		text = strings.TrimSpace(text)
		if isNoise(text) || len(text) < 10 {
			continue
		}

		if len(text) > 500 {
			text = text[:500] + "..."
		}
		fmt.Fprintf(&sb, "[%s] %s\n", role, text)
	}

	return sb.String()
}

func extractText(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}

	return ""
}

func isNoise(text string) bool {
	markers := []string{
		"<command-message>",
		"<command-name>",
		"Base directory for this skill",
		"<system-reminder>",
	}
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}
