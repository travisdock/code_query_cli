package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Tool definitions for OpenAI function calling
var ToolDefinitions = []map[string]interface{}{
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "ls",
			"description": "List directory contents. Use this to see what files and folders exist in a directory.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path to list (default: current directory)",
					},
				},
				"required": []string{},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "cat",
			"description": "Read and display the entire contents of a file.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"path"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "head",
			"description": "Read the first N lines of a file. Useful for previewing large files.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
					"lines": map[string]interface{}{
						"type":        "integer",
						"description": "Number of lines to read (default: 50)",
					},
				},
				"required": []string{"path"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "grep",
			"description": "Search for a pattern in files. Returns matching lines with file names and line numbers.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "The search pattern (regular expression)",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File or directory to search in (default: current directory)",
					},
					"recursive": map[string]interface{}{
						"type":        "boolean",
						"description": "Search recursively in subdirectories (default: true)",
					},
				},
				"required": []string{"pattern"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "find",
			"description": "Find files by name pattern. Searches for files matching the given pattern.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "File name pattern to search for (e.g., '*.go', 'config*')",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory to search in (default: current directory)",
					},
				},
				"required": []string{"pattern"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "tree",
			"description": "Show directory structure as a tree. Useful for understanding project layout.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Root directory (default: current directory)",
					},
					"depth": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum depth to display (default: 3)",
					},
				},
				"required": []string{},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "write_markdown",
			"description": "Create a new markdown (.md) file with the provided content. Use this to create documentation, READMEs, or reports based on information gathered from the codebase.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path where the markdown file should be created (must end with .md)",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The markdown content to write to the file",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	},
}

// ExecuteTool runs a tool and returns its output
func ExecuteTool(name string, argsJSON string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		PrintError(fmt.Sprintf("Failed to parse tool arguments: %v", err))
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	// Validate and sanitize paths
	if path, ok := args["path"].(string); ok {
		if _, err := validatePath(path); err != nil {
			return "", err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch name {
	case "ls":
		return executeLs(ctx, args)
	case "cat":
		return executeCat(ctx, args)
	case "head":
		return executeHead(ctx, args)
	case "grep":
		return executeGrep(ctx, args)
	case "find":
		return executeFind(ctx, args)
	case "tree":
		return executeTree(ctx, args)
	case "write_markdown":
		return executeWriteMarkdown(ctx, args)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func validatePath(path string) (string, error) {
	// Prevent path traversal
	clean := filepath.Clean(path)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		// Allow absolute paths within cwd
		cwd, err := filepath.Abs(".")
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %v", err)
		}
		abs, err := filepath.Abs(clean)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path: %v", err)
		}
		if !strings.HasPrefix(abs, cwd) {
			return "", fmt.Errorf("path traversal not allowed: %s", path)
		}
	}
	return clean, nil
}

func getString(args map[string]interface{}, key, defaultVal string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

func getInt(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

func getBool(args map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultVal
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	result := string(output)

	// Truncate very long outputs
	const maxLen = 50000
	if len(result) > maxLen {
		result = result[:maxLen] + "\n... (output truncated)"
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out")
		}
		// Return output even on error (grep returns 1 for no matches)
		if result != "" {
			return result, nil
		}
		return "", err
	}
	return result, nil
}

func executeLs(ctx context.Context, args map[string]interface{}) (string, error) {
	path := getString(args, "path", ".")
	return runCommand(ctx, "ls", "-la", path)
}

func executeCat(ctx context.Context, args map[string]interface{}) (string, error) {
	path := getString(args, "path", "")
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if IsPathBlocked(path) {
		return "", fmt.Errorf("access denied: %s is in ignore list", path)
	}
	return runCommand(ctx, "cat", path)
}

func executeHead(ctx context.Context, args map[string]interface{}) (string, error) {
	path := getString(args, "path", "")
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if IsPathBlocked(path) {
		return "", fmt.Errorf("access denied: %s is in ignore list", path)
	}
	lines := getInt(args, "lines", 50)
	return runCommand(ctx, "head", "-n", fmt.Sprintf("%d", lines), path)
}

func executeGrep(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern := getString(args, "pattern", "")
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	path := getString(args, "path", ".")
	recursive := getBool(args, "recursive", true)

	grepArgs := []string{"-n", "--color=never"}
	if recursive {
		grepArgs = append(grepArgs, "-r")
	}
	// Use "--" to separate options from pattern to prevent injection
	// (e.g., pattern "-e malicious" being interpreted as a flag)
	grepArgs = append(grepArgs, "--", pattern, path)

	result, err := runCommand(ctx, "grep", grepArgs...)
	if err != nil {
		return result, err
	}

	// Filter out results from blocked files
	var filtered []string
	for _, line := range strings.Split(result, "\n") {
		// Grep output format: "filename:linenum:content" or "filename:content"
		if idx := strings.Index(line, ":"); idx > 0 {
			filename := line[:idx]
			if IsPathBlocked(filename) {
				continue
			}
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n"), nil
}

func executeFind(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern := getString(args, "pattern", "")
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	path := getString(args, "path", ".")

	result, err := runCommand(ctx, "find", path, "-name", pattern, "-type", "f")
	if err != nil {
		return result, err
	}

	// Filter out blocked files
	var filtered []string
	for _, line := range strings.Split(result, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || IsPathBlocked(line) {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n"), nil
}

func executeTree(ctx context.Context, args map[string]interface{}) (string, error) {
	path := getString(args, "path", ".")
	depth := getInt(args, "depth", 3)

	// Try tree command first, fall back to find if not available
	result, err := runCommand(ctx, "tree", "-L", fmt.Sprintf("%d", depth), path)
	if err != nil {
		// Fallback: use find to simulate tree
		return runCommand(ctx, "find", path, "-maxdepth", fmt.Sprintf("%d", depth), "-print")
	}
	return result, nil
}

func executeWriteMarkdown(ctx context.Context, args map[string]interface{}) (string, error) {
	path := getString(args, "path", "")
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Validate that the file ends with .md
	if !strings.HasSuffix(strings.ToLower(path), ".md") {
		return "", fmt.Errorf("only markdown files (.md) can be created")
	}

	content := getString(args, "content", "")
	if content == "" {
		return "", fmt.Errorf("content is required")
	}

	// Format the markdown content to remove excessive whitespace
	formattedContent := formatMarkdown(content)

	// Validate path for security and get cleaned path
	clean, err := validatePath(path)
	if err != nil {
		return "", err
	}

	// Check if file already exists
	if _, err := os.Stat(clean); err == nil {
		return "", fmt.Errorf("file already exists: %s", path)
	}

	// Create parent directories if needed
	dir := filepath.Dir(clean)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	// Write the file
	if err := os.WriteFile(clean, []byte(formattedContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return fmt.Sprintf("Successfully created markdown file: %s", path), nil
}

// formatMarkdown cleans up markdown content by:
// - Normalizing line endings to \n
// - Limiting consecutive blank lines to a maximum of 2
// - Trimming trailing whitespace from lines
// - Ensuring file ends with a single newline
func formatMarkdown(content string) string {
	// Normalize line endings to \n
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	var formatted []string
	blankLineCount := 0

	for _, line := range lines {
		// Trim trailing whitespace from each line
		trimmed := strings.TrimRight(line, " \t")

		// Track consecutive blank lines
		if trimmed == "" {
			blankLineCount++
			// Limit consecutive blank lines to maximum of 2
			if blankLineCount <= 2 {
				formatted = append(formatted, "")
			}
		} else {
			blankLineCount = 0
			formatted = append(formatted, trimmed)
		}
	}

	// Join lines and ensure file ends with single newline
	result := strings.Join(formatted, "\n")
	result = strings.TrimRight(result, "\n") + "\n"

	return result
}

// FormatToolCall returns a human-readable string for displaying a tool call
func FormatToolCall(name string, argsJSON string) string {
	var args map[string]interface{}
	json.Unmarshal([]byte(argsJSON), &args)

	switch name {
	case "ls":
		path := getString(args, "path", ".")
		return path
	case "cat", "head":
		path := getString(args, "path", "")
		if lines := getInt(args, "lines", 0); lines > 0 {
			return fmt.Sprintf("%s -n %d", path, lines)
		}
		return path
	case "grep":
		pattern := getString(args, "pattern", "")
		path := getString(args, "path", ".")
		if getBool(args, "recursive", true) {
			return fmt.Sprintf("-r \"%s\" %s", pattern, path)
		}
		return fmt.Sprintf("\"%s\" %s", pattern, path)
	case "find":
		pattern := getString(args, "pattern", "")
		path := getString(args, "path", ".")
		return fmt.Sprintf("\"%s\" %s", pattern, path)
	case "tree":
		path := getString(args, "path", ".")
		depth := getInt(args, "depth", 3)
		return fmt.Sprintf("-L %d %s", depth, path)
	case "write_markdown":
		path := getString(args, "path", "")
		return path
	default:
		return argsJSON
	}
}
