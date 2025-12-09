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

When answering questions:
1. First explore the codebase structure using ls or tree
2. Use find to locate specific files by name
3. Use grep to search for patterns in code
4. Use cat or head to read file contents
5. Provide clear, concise answers based on what you find

Always use the tools to verify your answers - don't guess about code you haven't read.`,
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
		return assistantMsg.Content, nil
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

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v\nBody: %s", err, string(body))
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return &chatResp, nil
}

// Reset clears conversation history (keeps system message)
func (c *Client) Reset() {
	c.messages = c.messages[:1]
}
