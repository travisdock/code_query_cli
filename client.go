package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Reasoning  string     `json:"reasoning,omitempty"` // Some models (o1, deepseek) use this field
}

// ToolCall represents a function call from the model
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatRequest is the request body for chat completions
type ChatRequest struct {
	Model    string                   `json:"model"`
	Messages []Message                `json:"messages"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
}

// ChatResponse is the response from chat completions
type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Client handles communication with OpenAI-compatible APIs
type Client struct {
	config   *Config
	http     *http.Client
	messages []Message
}

// NewClient creates a new API client
func NewClient(cfg *Config) *Client {
	return &Client{
		config: cfg,
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
		messages: []Message{
			{
				Role: "system",
				Content: `You are a helpful assistant that answers questions about codebases.
You have access to tools that let you explore the file system: ls, cat, head, grep, find, and tree.
You can also create markdown documentation files using the write_markdown tool.

IMPORTANT: You MUST use the tool calling feature to invoke tools. Do NOT write JSON or function calls in your response text. Use the tool_calls mechanism provided by the API.

When answering questions:
1. Use grep to search for specific patterns or keywords in code
2. Use find to locate files by name pattern
3. Use cat or head to read file contents
4. Use ls or tree to explore directory structure
5. After gathering information, provide a clear, concise answer
6. Use write_markdown to create documentation files when requested (prefer current directory)

Make a step by step plan of what tools you will use and why before starting tool executions.

## Example 1: Answering a Question

**User question:** "Where is the database connection configured?"

**Reasoning:** The user wants to find database configuration. This could be in config files, environment handling code, or database initialization. I should search for database-related keywords first, then read the relevant files.

**Plan:**
1. Use grep to search for "database" or "db" patterns to find relevant files
2. Use cat to read the most promising file(s)
3. Summarize the findings

**Tool calls:**
1. grep({"pattern": "database|db.*connect", "path": ".", "recursive": true})
   → Found matches in config.go:15 and db/client.go:23

2. cat({"path": "config.go"})
   → Shows Config struct with DatabaseURL field and LoadConfig function reading from environment

**Answer:** The database connection is configured in config.go. The Config struct (line 12) has a DatabaseURL field that gets populated from the DATABASE_URL environment variable in the LoadConfig function (line 25). The actual connection is established in db/client.go using this config value.

## Example 2: Creating Documentation

**User request:** "Review the authentication code and create a README documenting how it works"

**Plan:**
1. Use find or grep to locate authentication-related files
2. Use cat to read the authentication implementation
3. Use write_markdown to create a README in the current directory

**Tool calls:**
1. grep({"pattern": "auth", "path": ".", "recursive": true})
   → Found auth.go, middleware.go
2. cat({"path": "auth.go"})
   → Read authentication logic
3. write_markdown({"path": "AUTH_GUIDE.md", "content": "# Authentication\\n\\nThis system uses JWT..."})
   → Created documentation file in current directory

**IMPORTANT for write_markdown:**
- DEFAULT: Use just a filename to write to current directory (e.g., "README.md", "API_DOCS.md")
- ONLY use a directory path if user explicitly asks for it (e.g., "put it in docs/")
- Directory must already exist - tool cannot create directories

---

Always use the tools to verify your answers - don't guess about code you haven't read.
When you have enough information, respond with your final answer in plain text.`,
			},
		},
	}
}

// ToolCallback is called for each tool execution with name, raw args JSON, and result
type ToolCallback func(name, argsJSON, result string)

// Chat sends a message and handles tool calls in a loop
func (c *Client) Chat(userMessage string, onToolCall ToolCallback) (string, error) {
	// Add user message to history
	c.messages = append(c.messages, Message{
		Role:    "user",
		Content: userMessage,
	})

	for {
		resp, err := c.sendRequest()
		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no response from model")
		}

		choice := resp.Choices[0]
		assistantMsg := choice.Message

		// Add assistant message to history
		c.messages = append(c.messages, assistantMsg)

		// If there are tool calls, execute them
		if len(assistantMsg.ToolCalls) > 0 {
			for _, tc := range assistantMsg.ToolCalls {
				// Execute the tool
				result, err := ExecuteTool(tc.Function.Name, tc.Function.Arguments)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}

				// Notify about tool call with result
				if onToolCall != nil {
					onToolCall(tc.Function.Name, tc.Function.Arguments, result)
				}

				// Add tool result to history
				c.messages = append(c.messages, Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
			// Continue the loop to get the next response
			continue
		}

		// No more tool calls, return the final response
		// Some reasoning models (o1, deepseek, etc.) put response in "reasoning" field
		response := assistantMsg.Content
		if response == "" && assistantMsg.Reasoning != "" {
			response = assistantMsg.Reasoning
		}
		return response, nil
	}
}

func (c *Client) sendRequest() (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:    c.config.Model,
		Messages: c.messages,
		Tools:    ToolDefinitions,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	if debugMode {
		fmt.Printf("[debug] Sending %d tools, %d messages\n", len(reqBody.Tools), len(reqBody.Messages))
	}

	url := strings.TrimSuffix(c.config.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Trim whitespace - some providers (OpenRouter) pad responses
	body = bytes.TrimSpace(body)

	// Check status code first (issue #3 from review)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	if debugMode {
		fmt.Printf("[debug] Raw API response: %s\n", string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v\nBody: %s", err, string(body))
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	return &chatResp, nil
}

// Reset clears conversation history (keeps system message)
func (c *Client) Reset() {
	c.messages = c.messages[:1]
}
