package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 30 * time.Second}

func cmdFetch(args []string) {
	from, to := parseDate(args)

	cfg, err := loadAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	userID, err := fetchUserID(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get user: %v\n", err)
		os.Exit(1)
	}

	allEntries := make([]json.RawMessage, 0)
	page := 1

	for {
		url := fmt.Sprintf(
			"https://api.harvestapp.com/v2/time_entries?user_id=%d&from=%s&to=%s&page=%d&per_page=100",
			userID, from, to, page,
		)

		body, err := harvestGet(cfg, url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch failed: %v\n", err)
			os.Exit(1)
		}

		var resp struct {
			TimeEntries []json.RawMessage `json:"time_entries"`
			TotalPages  int               `json:"total_pages"`
			Page        int               `json:"page"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse response: %v\n", err)
			os.Exit(1)
		}

		allEntries = append(allEntries, resp.TimeEntries...)

		if resp.Page >= resp.TotalPages {
			break
		}
		page++
	}

	out, _ := json.MarshalIndent(allEntries, "", "  ")
	fmt.Println(string(out))
}

func fetchUserID(cfg *AuthConfig) (int, error) {
	body, err := harvestGet(cfg, "https://api.harvestapp.com/v2/users/me")
	if err != nil {
		return 0, err
	}
	var user struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return 0, err
	}
	return user.ID, nil
}

func cmdUpdateNotes(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: harvest-cli update-notes <entry_id> <new_notes>")
		os.Exit(1)
	}

	entryID := args[0]
	newNotes := args[1]

	// Validate entry ID is numeric to prevent URL injection
	for _, c := range entryID {
		if c < '0' || c > '9' {
			fmt.Fprintf(os.Stderr, "invalid entry ID: %s (must be numeric)\n", entryID)
			os.Exit(1)
		}
	}

	cfg, err := loadAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	url := fmt.Sprintf("https://api.harvestapp.com/v2/time_entries/%s", entryID)
	payload := fmt.Sprintf(`{"notes":%s}`, mustJSON(newNotes))

	req, err := http.NewRequest(http.MethodPatch, url, strings.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create request: %v\n", err)
		os.Exit(1)
	}
	setHarvestHeaders(req, cfg)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "update failed (%d): %s\n", resp.StatusCode, body)
		os.Exit(1)
	}

	fmt.Printf("Updated entry %s\n", entryID)
}


func parseDate(args []string) (from, to string) {
	today := time.Now().Format("2006-01-02")
	from = today
	to = today

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--date":
			if i+1 < len(args) {
				from = args[i+1]
				to = args[i+1]
				i++
			}
		case "--from":
			if i+1 < len(args) {
				from = args[i+1]
				i++
			}
		case "--to":
			if i+1 < len(args) {
				to = args[i+1]
				i++
			}
		}
	}

	if _, err := time.Parse("2006-01-02", from); err != nil {
		fmt.Fprintf(os.Stderr, "invalid from date %q: expected YYYY-MM-DD\n", from)
		os.Exit(1)
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		fmt.Fprintf(os.Stderr, "invalid to date %q: expected YYYY-MM-DD\n", to)
		os.Exit(1)
	}

	return from, to
}

func harvestGet(cfg *AuthConfig, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setHarvestHeaders(req, cfg)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	return body, nil
}

func setHarvestHeaders(req *http.Request, cfg *AuthConfig) {
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	req.Header.Set("Harvest-Account-Id", cfg.AccountID)
	req.Header.Set("User-Agent", "HarvestAnnotateSkill")
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
