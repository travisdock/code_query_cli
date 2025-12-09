package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4o",
	}

	// Try to load from config file first
	configPath := getConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			PrintError(fmt.Sprintf("Failed to parse config file %s: %v", configPath, err))
		}
	}

	// Environment variables override config file
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg.APIKey = key
	}
	if url := os.Getenv("OPENAI_BASE_URL"); url != "" {
		cfg.BaseURL = url
	}
	if model := os.Getenv("CODEQUERY_MODEL"); model != "" {
		cfg.Model = model
	}

	return cfg, nil
}

func getConfigPath() string {
	// Check XDG_CONFIG_HOME first
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "codequery", "config.json")
	}
	// Fall back to ~/.config
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "codequery", "config.json")
}
