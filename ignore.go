package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Default patterns that are always blocked
var defaultBlockedPatterns = []string{
	".env",
	".env.*",
	"*.pem",
	"*.key",
	"*.p12",
	"*.pfx",
	"*.secret",
	"*credentials*",
	"*secret*",
	".aws/credentials",
	".ssh/*",
	"id_rsa",
	"id_ed25519",
	"*.keystore",
	".netrc",
	".npmrc",
	".pypirc",
}

var blockedPatterns []string

// LoadIgnorePatterns loads patterns from .codequeryignore and combines with defaults
func LoadIgnorePatterns() {
	blockedPatterns = append(blockedPatterns, defaultBlockedPatterns...)

	// Try to load .codequeryignore from current directory
	file, err := os.Open(".codequeryignore")
	if err != nil {
		return // File doesn't exist, just use defaults
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		blockedPatterns = append(blockedPatterns, line)
	}
}

// IsPathBlocked checks if a path matches any blocked pattern
func IsPathBlocked(path string) bool {
	// Normalize the path
	path = filepath.Clean(path)
	base := filepath.Base(path)

	for _, pattern := range blockedPatterns {
		// Check against full path
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// Check against basename
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		// Check if pattern is contained in path (for patterns like "*secret*")
		if strings.Contains(pattern, "*") {
			if matched, _ := filepath.Match(pattern, base); matched {
				return true
			}
		} else {
			// Exact match or suffix match for non-glob patterns
			if base == pattern || strings.HasSuffix(path, "/"+pattern) {
				return true
			}
		}
	}
	return false
}

// FilterBlockedPaths removes blocked paths from a list
func FilterBlockedPaths(paths []string) []string {
	var filtered []string
	for _, p := range paths {
		if !IsPathBlocked(p) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
