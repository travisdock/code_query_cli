package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_BASE_URL")
	os.Unsetenv("CODEQUERY_MODEL")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %v, want %v", cfg.BaseURL, "https://api.openai.com/v1")
	}
	if cfg.Model != "gpt-4o" {
		t.Errorf("Model = %v, want %v", cfg.Model, "gpt-4o")
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("OPENAI_API_KEY", "test-api-key")
	os.Setenv("OPENAI_BASE_URL", "https://custom.api.com/v1")
	os.Setenv("CODEQUERY_MODEL", "gpt-3.5-turbo")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("OPENAI_BASE_URL")
		os.Unsetenv("CODEQUERY_MODEL")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.APIKey != "test-api-key" {
		t.Errorf("APIKey = %v, want %v", cfg.APIKey, "test-api-key")
	}
	if cfg.BaseURL != "https://custom.api.com/v1" {
		t.Errorf("BaseURL = %v, want %v", cfg.BaseURL, "https://custom.api.com/v1")
	}
	if cfg.Model != "gpt-3.5-turbo" {
		t.Errorf("Model = %v, want %v", cfg.Model, "gpt-3.5-turbo")
	}
}

func TestGetConfigPath_Default(t *testing.T) {
	os.Unsetenv("XDG_CONFIG_HOME")

	path := getConfigPath()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "codequery", "config.json")

	if path != expected {
		t.Errorf("getConfigPath() = %v, want %v", path, expected)
	}
}

func TestGetConfigPath_XDG(t *testing.T) {
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	defer os.Unsetenv("XDG_CONFIG_HOME")

	path := getConfigPath()
	expected := "/custom/config/codequery/config.json"

	if path != expected {
		t.Errorf("getConfigPath() = %v, want %v", path, expected)
	}
}
