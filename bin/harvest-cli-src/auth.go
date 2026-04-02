package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AuthConfig struct {
	AccessToken string `json:"access_token"`
	AccountID   string `json:"account_id"`
}

func authFilePath() string {
	if dir := os.Getenv("HARVEST_DATA_DIR"); dir != "" {
		return filepath.Join(dir, "auth.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".claude", "auth.json")
}

func loadAuth() (*AuthConfig, error) {
	path := authFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			placeholder := []byte(`{"access_token": "...", "account_id": "..."}`)
			if mkErr := os.MkdirAll(filepath.Dir(path), 0700); mkErr != nil {
				return nil, fmt.Errorf("cannot create directory for %s: %w", path, mkErr)
			}
			if wErr := os.WriteFile(path, placeholder, 0600); wErr != nil {
				return nil, fmt.Errorf("cannot create placeholder auth config at %s: %w", path, wErr)
			}
			return nil, fmt.Errorf("auth not configured — edit %s with your Harvest credentials", path)
		}
		return nil, fmt.Errorf("cannot read auth config at %s: %w", path, err)
	}
	var cfg AuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid auth config: %w", err)
	}
	if cfg.AccessToken == "" || cfg.AccessToken == "..." {
		return nil, fmt.Errorf("auth not configured — edit %s with your Harvest credentials", path)
	}
	return &cfg, nil
}
