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
		return nil, fmt.Errorf("no auth config found at %s", path)
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
