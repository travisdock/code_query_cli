package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidatePath_Safe(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"current dir", "."},
		{"relative path", "src/main.go"},
		{"nested path", "foo/bar/baz.txt"},
		{"file only", "file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePath(tt.path); err != nil {
				t.Errorf("validatePath(%q) = %v, want nil", tt.path, err)
			}
		})
	}
}

func TestValidatePath_Traversal(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"parent dir", ".."},
		{"parent traversal", "../etc/passwd"},
		{"nested traversal", "foo/../../bar"},
		{"absolute path", "/etc/passwd"},
		{"absolute home", "/home/user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if err == nil {
				t.Errorf("validatePath(%q) = nil, want error", tt.path)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	args := map[string]interface{}{
		"name":  "test",
		"empty": "",
		"num":   42.0,
	}

	tests := []struct {
		key      string
		def      string
		expected string
	}{
		{"name", "default", "test"},
		{"empty", "default", "default"}, // empty string returns default
		{"missing", "default", "default"},
		{"num", "default", "default"}, // wrong type returns default
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := getString(args, tt.key, tt.def); got != tt.expected {
				t.Errorf("getString(args, %q, %q) = %q, want %q", tt.key, tt.def, got, tt.expected)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	args := map[string]interface{}{
		"count":  42.0, // JSON numbers are float64
		"zero":   0.0,
		"string": "not a number",
	}

	tests := []struct {
		key      string
		def      int
		expected int
	}{
		{"count", 10, 42},
		{"zero", 10, 0},
		{"missing", 10, 10},
		{"string", 10, 10}, // wrong type returns default
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := getInt(args, tt.key, tt.def); got != tt.expected {
				t.Errorf("getInt(args, %q, %d) = %d, want %d", tt.key, tt.def, got, tt.expected)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	args := map[string]interface{}{
		"enabled":  true,
		"disabled": false,
		"string":   "true",
	}

	tests := []struct {
		key      string
		def      bool
		expected bool
	}{
		{"enabled", false, true},
		{"disabled", true, false},
		{"missing", true, true},
		{"string", false, false}, // wrong type returns default
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := getBool(args, tt.key, tt.def); got != tt.expected {
				t.Errorf("getBool(args, %q, %v) = %v, want %v", tt.key, tt.def, got, tt.expected)
			}
		})
	}
}

func TestExecuteTool_InvalidJSON(t *testing.T) {
	_, err := ExecuteTool("ls", "not valid json")
	if err == nil {
		t.Error("ExecuteTool with invalid JSON should return error")
	}
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	_, err := ExecuteTool("unknown", "{}")
	if err == nil {
		t.Error("ExecuteTool with unknown tool should return error")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("Error message should contain 'unknown tool', got: %v", err)
	}
}

func TestExecuteTool_PathTraversal(t *testing.T) {
	_, err := ExecuteTool("cat", `{"path": "../../../etc/passwd"}`)
	if err == nil {
		t.Error("ExecuteTool with path traversal should return error")
	}
}

func TestExecuteTool_Ls(t *testing.T) {
	result, err := ExecuteTool("ls", `{"path": "."}`)
	if err != nil {
		t.Fatalf("ExecuteTool ls error: %v", err)
	}
	if !strings.Contains(result, "go.mod") && !strings.Contains(result, "main.go") {
		t.Errorf("ls output should contain project files, got: %s", result)
	}
}

func TestExecuteTool_Cat(t *testing.T) {
	// Create a temporary test file
	content := "test content\nline 2"
	tmpFile := filepath.Join(os.TempDir(), "codequery_test.txt")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpFile)

	// We need to use a relative path for the test
	testFile := "test_cat_file.txt"
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	result, err := ExecuteTool("cat", `{"path": "test_cat_file.txt"}`)
	if err != nil {
		t.Fatalf("ExecuteTool cat error: %v", err)
	}
	if result != content {
		t.Errorf("cat output = %q, want %q", result, content)
	}
}

func TestExecuteTool_Cat_MissingPath(t *testing.T) {
	_, err := ExecuteTool("cat", `{}`)
	if err == nil {
		t.Error("cat without path should return error")
	}
}

func TestExecuteTool_Head(t *testing.T) {
	// Create a test file with multiple lines
	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	testFile := "test_head_file.txt"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	result, err := ExecuteTool("head", `{"path": "test_head_file.txt", "lines": 2}`)
	if err != nil {
		t.Fatalf("ExecuteTool head error: %v", err)
	}
	expected := "line 1\nline 2\n"
	if result != expected {
		t.Errorf("head output = %q, want %q", result, expected)
	}
}

func TestExecuteTool_Grep(t *testing.T) {
	// Create a test file
	content := "func main() {\nfmt.Println(\"hello\")\n}\n"
	testFile := "test_grep_file.txt"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	result, err := ExecuteTool("grep", `{"pattern": "main", "path": "test_grep_file.txt", "recursive": false}`)
	if err != nil {
		t.Fatalf("ExecuteTool grep error: %v", err)
	}
	if !strings.Contains(result, "func main") {
		t.Errorf("grep output should contain match, got: %s", result)
	}
}

func TestExecuteTool_Grep_MissingPattern(t *testing.T) {
	_, err := ExecuteTool("grep", `{"path": "."}`)
	if err == nil {
		t.Error("grep without pattern should return error")
	}
}

func TestExecuteTool_Find(t *testing.T) {
	result, err := ExecuteTool("find", `{"pattern": "*.go", "path": "."}`)
	if err != nil {
		t.Fatalf("ExecuteTool find error: %v", err)
	}
	if !strings.Contains(result, "main.go") {
		t.Errorf("find output should contain main.go, got: %s", result)
	}
}

func TestExecuteTool_Find_MissingPattern(t *testing.T) {
	_, err := ExecuteTool("find", `{"path": "."}`)
	if err == nil {
		t.Error("find without pattern should return error")
	}
}

func TestExecuteTool_Tree(t *testing.T) {
	result, err := ExecuteTool("tree", `{"path": ".", "depth": 1}`)
	if err != nil {
		t.Fatalf("ExecuteTool tree error: %v", err)
	}
	// Tree or find fallback should produce some output
	if result == "" {
		t.Error("tree output should not be empty")
	}
}

func TestFormatToolCall_Ls(t *testing.T) {
	result := FormatToolCall("ls", `{"path": "src"}`)
	if result != "src" {
		t.Errorf("FormatToolCall(ls) = %q, want %q", result, "src")
	}
}

func TestFormatToolCall_LsDefault(t *testing.T) {
	result := FormatToolCall("ls", `{}`)
	if result != "." {
		t.Errorf("FormatToolCall(ls default) = %q, want %q", result, ".")
	}
}

func TestFormatToolCall_Cat(t *testing.T) {
	result := FormatToolCall("cat", `{"path": "main.go"}`)
	if result != "main.go" {
		t.Errorf("FormatToolCall(cat) = %q, want %q", result, "main.go")
	}
}

func TestFormatToolCall_Head(t *testing.T) {
	result := FormatToolCall("head", `{"path": "file.txt", "lines": 10}`)
	expected := "file.txt -n 10"
	if result != expected {
		t.Errorf("FormatToolCall(head) = %q, want %q", result, expected)
	}
}

func TestFormatToolCall_HeadNoLines(t *testing.T) {
	result := FormatToolCall("head", `{"path": "file.txt"}`)
	if result != "file.txt" {
		t.Errorf("FormatToolCall(head no lines) = %q, want %q", result, "file.txt")
	}
}

func TestFormatToolCall_Grep(t *testing.T) {
	result := FormatToolCall("grep", `{"pattern": "TODO", "path": "src", "recursive": true}`)
	expected := `-r "TODO" src`
	if result != expected {
		t.Errorf("FormatToolCall(grep) = %q, want %q", result, expected)
	}
}

func TestFormatToolCall_GrepNonRecursive(t *testing.T) {
	result := FormatToolCall("grep", `{"pattern": "main", "path": ".", "recursive": false}`)
	expected := `"main" .`
	if result != expected {
		t.Errorf("FormatToolCall(grep non-recursive) = %q, want %q", result, expected)
	}
}

func TestFormatToolCall_Find(t *testing.T) {
	result := FormatToolCall("find", `{"pattern": "*.go", "path": "src"}`)
	expected := `"*.go" src`
	if result != expected {
		t.Errorf("FormatToolCall(find) = %q, want %q", result, expected)
	}
}

func TestFormatToolCall_Tree(t *testing.T) {
	result := FormatToolCall("tree", `{"path": ".", "depth": 2}`)
	expected := "-L 2 ."
	if result != expected {
		t.Errorf("FormatToolCall(tree) = %q, want %q", result, expected)
	}
}

func TestFormatToolCall_Unknown(t *testing.T) {
	argsJSON := `{"foo": "bar"}`
	result := FormatToolCall("unknown", argsJSON)
	if result != argsJSON {
		t.Errorf("FormatToolCall(unknown) = %q, want %q", result, argsJSON)
	}
}

func TestRunCommand_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Use sleep command to test timeout
	_, err := runCommand(ctx, "sleep", "10")
	if err == nil {
		t.Error("runCommand with short timeout should return error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Error should indicate timeout, got: %v", err)
	}
}

// Tests for write_markdown tool
func TestExecuteTool_WriteMarkdown_Success(t *testing.T) {
	testFile := "test_write_markdown.md"
	defer os.Remove(testFile)

	args := `{"path": "test_write_markdown.md", "content": "# Test\n\nContent"}`
	result, err := ExecuteTool("write_markdown", args)
	if err != nil {
		t.Fatalf("ExecuteTool write_markdown error: %v", err)
	}

	if !strings.Contains(result, "Successfully created") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file was created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	expected := "# Test\n\nContent\n"
	if string(content) != expected {
		t.Errorf("File content = %q, want %q", string(content), expected)
	}
}

func TestExecuteTool_WriteMarkdown_WithSubdirectory(t *testing.T) {
	testDir := "test_write_markdown_dir"
	testFile := filepath.Join(testDir, "guide.md")
	defer os.RemoveAll(testDir)

	args := fmt.Sprintf(`{"path": "%s", "content": "# Guide\n\nSteps"}`, testFile)
	result, err := ExecuteTool("write_markdown", args)
	if err != nil {
		t.Fatalf("ExecuteTool write_markdown error: %v", err)
	}

	if !strings.Contains(result, "Successfully created") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify directory and file were created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should have been created")
	}
}

func TestExecuteTool_WriteMarkdown_NonMarkdownExtension(t *testing.T) {
	testFile := "test_write_markdown.txt"

	args := fmt.Sprintf(`{"path": "%s", "content": "content"}`, testFile)
	_, err := ExecuteTool("write_markdown", args)
	if err == nil {
		t.Error("write_markdown should reject non-.md files")
	}
	if !strings.Contains(err.Error(), "only markdown files") {
		t.Errorf("Error should mention markdown restriction, got: %v", err)
	}
}

func TestExecuteTool_WriteMarkdown_FileExists(t *testing.T) {
	testFile := "test_write_markdown_exists.md"

	// Create existing file
	err := os.WriteFile(testFile, []byte("existing content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	args := fmt.Sprintf(`{"path": "%s", "content": "new content"}`, testFile)
	_, err = ExecuteTool("write_markdown", args)
	if err == nil {
		t.Error("write_markdown should reject overwriting existing files")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error should mention file exists, got: %v", err)
	}
}

func TestExecuteTool_WriteMarkdown_MissingPath(t *testing.T) {
	_, err := ExecuteTool("write_markdown", `{"content": "test"}`)
	if err == nil {
		t.Error("write_markdown without path should return error")
	}
}

func TestExecuteTool_WriteMarkdown_MissingContent(t *testing.T) {
	_, err := ExecuteTool("write_markdown", `{"path": "test.md"}`)
	if err == nil {
		t.Error("write_markdown without content should return error")
	}
}

func TestExecuteTool_WriteMarkdown_PathTraversal(t *testing.T) {
	_, err := ExecuteTool("write_markdown", `{"path": "../../../etc/test.md", "content": "malicious"}`)
	if err == nil {
		t.Error("write_markdown with path traversal should return error")
	}
}

func TestFormatMarkdown_RemoveExcessiveBlankLines(t *testing.T) {
	input := "# Title\n\n\n\n\nContent\n\n\n\nMore"
	expected := "# Title\n\n\nContent\n\n\nMore\n"
	result := formatMarkdown(input)
	if result != expected {
		t.Errorf("formatMarkdown() = %q, want %q", result, expected)
	}
}

func TestFormatMarkdown_TrimTrailingSpaces(t *testing.T) {
	input := "# Title   \n\nContent with spaces   \n"
	expected := "# Title\n\nContent with spaces\n"
	result := formatMarkdown(input)
	if result != expected {
		t.Errorf("formatMarkdown() = %q, want %q", result, expected)
	}
}

func TestFormatMarkdown_NormalizeLineEndings(t *testing.T) {
	input := "Title\r\nContent\rMore"
	expected := "Title\nContent\nMore\n"
	result := formatMarkdown(input)
	if result != expected {
		t.Errorf("formatMarkdown() = %q, want %q", result, expected)
	}
}

func TestFormatMarkdown_EnsureSingleNewlineAtEnd(t *testing.T) {
	input := "# Title\nContent"
	expected := "# Title\nContent\n"
	result := formatMarkdown(input)
	if result != expected {
		t.Errorf("formatMarkdown() = %q, want %q", result, expected)
	}
}

func TestFormatMarkdown_RemoveMultipleNewlinesAtEnd(t *testing.T) {
	input := "# Title\nContent\n\n\n\n"
	expected := "# Title\nContent\n"
	result := formatMarkdown(input)
	if result != expected {
		t.Errorf("formatMarkdown() = %q, want %q", result, expected)
	}
}

func TestFormatToolCall_WriteMarkdown(t *testing.T) {
	result := FormatToolCall("write_markdown", `{"path": "docs/README.md", "content": "test"}`)
	expected := "docs/README.md"
	if result != expected {
		t.Errorf("FormatToolCall(write_markdown) = %q, want %q", result, expected)
	}
}
