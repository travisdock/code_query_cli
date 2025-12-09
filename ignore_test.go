package main

import (
	"os"
	"testing"
)

func init() {
	// Ensure patterns are loaded for tests
	LoadIgnorePatterns()
}

func TestIsPathBlocked_EnvFiles(t *testing.T) {
	tests := []struct {
		path    string
		blocked bool
	}{
		{".env", true},
		{".env.local", true},
		{".env.production", true},
		{"config/.env", true},
		{"src/app.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsPathBlocked(tt.path); got != tt.blocked {
				t.Errorf("IsPathBlocked(%q) = %v, want %v", tt.path, got, tt.blocked)
			}
		})
	}
}

func TestIsPathBlocked_KeyFiles(t *testing.T) {
	tests := []struct {
		path    string
		blocked bool
	}{
		{"server.pem", true},
		{"private.key", true},
		{"cert.p12", true},
		{"keystore.pfx", true},
		{"app.secret", true},
		{"id_rsa", true},
		{"id_ed25519", true},
		{".ssh/config", true},
		{".ssh/known_hosts", true},
		{"public_key.pub", false}, // .pub is not blocked
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsPathBlocked(tt.path); got != tt.blocked {
				t.Errorf("IsPathBlocked(%q) = %v, want %v", tt.path, got, tt.blocked)
			}
		})
	}
}

func TestIsPathBlocked_CredentialFiles(t *testing.T) {
	tests := []struct {
		path    string
		blocked bool
	}{
		{"credentials.json", true},
		{"aws_credentials", true},
		{".aws/credentials", true},
		{"secret.yaml", true},
		{"secrets.json", true},
		{".netrc", true},
		{".npmrc", true},
		{".pypirc", true},
		{"app.keystore", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsPathBlocked(tt.path); got != tt.blocked {
				t.Errorf("IsPathBlocked(%q) = %v, want %v", tt.path, got, tt.blocked)
			}
		})
	}
}

func TestIsPathBlocked_SafeFiles(t *testing.T) {
	tests := []struct {
		path    string
		blocked bool
	}{
		{"main.go", false},
		{"README.md", false},
		{"config.yaml", false},
		{"package.json", false},
		{"src/utils/helper.ts", false},
		{".gitignore", false},
		{"Dockerfile", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsPathBlocked(tt.path); got != tt.blocked {
				t.Errorf("IsPathBlocked(%q) = %v, want %v", tt.path, got, tt.blocked)
			}
		})
	}
}

func TestFilterBlockedPaths(t *testing.T) {
	input := []string{
		"main.go",
		".env",
		"config.yaml",
		"secrets.json",
		"README.md",
		"private.key",
	}

	result := FilterBlockedPaths(input)

	expected := []string{"main.go", "config.yaml", "README.md"}
	if len(result) != len(expected) {
		t.Errorf("FilterBlockedPaths() returned %d items, want %d", len(result), len(expected))
		t.Errorf("Got: %v", result)
		return
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("FilterBlockedPaths()[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

func TestFilterBlockedPaths_EmptyInput(t *testing.T) {
	result := FilterBlockedPaths([]string{})
	if result != nil && len(result) != 0 {
		t.Errorf("FilterBlockedPaths(empty) = %v, want empty slice", result)
	}
}

func TestFilterBlockedPaths_AllBlocked(t *testing.T) {
	input := []string{".env", "private.key", "secrets.json"}
	result := FilterBlockedPaths(input)
	if result != nil && len(result) != 0 {
		t.Errorf("FilterBlockedPaths(all blocked) = %v, want empty slice", result)
	}
}

func TestLoadIgnorePatterns_CustomFile(t *testing.T) {
	// Create a temporary .codequeryignore file
	content := `# Custom ignore patterns
*.log
temp/
# Another comment
debug_*.txt
`
	err := os.WriteFile(".codequeryignore", []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test ignore file: %v", err)
	}
	defer os.Remove(".codequeryignore")

	// Reset and reload patterns
	blockedPatterns = nil
	LoadIgnorePatterns()

	// Test custom patterns are loaded
	tests := []struct {
		path    string
		blocked bool
	}{
		{"app.log", true},
		{"debug_output.txt", true},
		{".env", true}, // default pattern still works
		{"main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsPathBlocked(tt.path); got != tt.blocked {
				t.Errorf("IsPathBlocked(%q) = %v, want %v", tt.path, got, tt.blocked)
			}
		})
	}
}
