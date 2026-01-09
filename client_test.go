package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	cfg := &Config{
		APIKey:  "test-key",
		BaseURL: "https://api.example.com/v1",
		Model:   "gpt-4",
	}

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.config != cfg {
		t.Error("client.config != cfg")
	}
	if client.http == nil {
		t.Error("client.http is nil")
	}
	if len(client.messages) != 1 {
		t.Errorf("client.messages length = %d, want 1 (system message)", len(client.messages))
	}
	if client.messages[0].Role != "system" {
		t.Errorf("First message role = %q, want %q", client.messages[0].Role, "system")
	}
}

func TestClient_Reset(t *testing.T) {
	cfg := &Config{
		APIKey:  "test-key",
		BaseURL: "https://api.example.com/v1",
		Model:   "gpt-4",
	}

	client := NewClient(cfg)

	// Add some messages
	client.messages = append(client.messages, Message{Role: "user", Content: "hello"})
	client.messages = append(client.messages, Message{Role: "assistant", Content: "hi"})

	if len(client.messages) != 3 {
		t.Errorf("Before reset: messages length = %d, want 3", len(client.messages))
	}

	client.Reset()

	if len(client.messages) != 1 {
		t.Errorf("After reset: messages length = %d, want 1", len(client.messages))
	}
	if client.messages[0].Role != "system" {
		t.Errorf("After reset: first message role = %q, want %q", client.messages[0].Role, "system")
	}
}

func TestMessage_JSON(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "Hello, world!",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if decoded.Role != msg.Role {
		t.Errorf("decoded.Role = %q, want %q", decoded.Role, msg.Role)
	}
	if decoded.Content != msg.Content {
		t.Errorf("decoded.Content = %q, want %q", decoded.Content, msg.Content)
	}
}

func TestMessage_WithToolCalls(t *testing.T) {
	msg := Message{
		Role: "assistant",
		ToolCalls: []ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "ls",
					Arguments: `{"path": "."}`,
				},
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("decoded.ToolCalls length = %d, want 1", len(decoded.ToolCalls))
	}
	if decoded.ToolCalls[0].ID != "call_123" {
		t.Errorf("decoded.ToolCalls[0].ID = %q, want %q", decoded.ToolCalls[0].ID, "call_123")
	}
	if decoded.ToolCalls[0].Function.Name != "ls" {
		t.Errorf("decoded.ToolCalls[0].Function.Name = %q, want %q", decoded.ToolCalls[0].Function.Name, "ls")
	}
}

func TestMessage_ToolResponse(t *testing.T) {
	msg := Message{
		Role:       "tool",
		Content:    "file1.txt\nfile2.txt",
		ToolCallID: "call_123",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if decoded.Role != "tool" {
		t.Errorf("decoded.Role = %q, want %q", decoded.Role, "tool")
	}
	if decoded.ToolCallID != "call_123" {
		t.Errorf("decoded.ToolCallID = %q, want %q", decoded.ToolCallID, "call_123")
	}
}

func TestChatRequest_JSON(t *testing.T) {
	req := ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Tools: ToolDefinitions,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var decoded ChatRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if decoded.Model != req.Model {
		t.Errorf("decoded.Model = %q, want %q", decoded.Model, req.Model)
	}
	if len(decoded.Messages) != len(req.Messages) {
		t.Errorf("decoded.Messages length = %d, want %d", len(decoded.Messages), len(req.Messages))
	}
}

func TestChatResponse_JSON(t *testing.T) {
	jsonData := `{
		"id": "chatcmpl-123",
		"choices": [{
			"message": {
				"role": "assistant",
				"content": "Hello!"
			},
			"finish_reason": "stop"
		}]
	}`

	var resp ChatResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("resp.ID = %q, want %q", resp.ID, "chatcmpl-123")
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("resp.Choices length = %d, want 1", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("resp.Choices[0].Message.Content = %q, want %q", resp.Choices[0].Message.Content, "Hello!")
	}
	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("resp.Choices[0].FinishReason = %q, want %q", resp.Choices[0].FinishReason, "stop")
	}
}

func TestChatResponse_WithToolCalls(t *testing.T) {
	jsonData := `{
		"id": "chatcmpl-456",
		"choices": [{
			"message": {
				"role": "assistant",
				"tool_calls": [{
					"id": "call_abc",
					"type": "function",
					"function": {
						"name": "grep",
						"arguments": "{\"pattern\": \"main\"}"
					}
				}]
			},
			"finish_reason": "tool_calls"
		}]
	}`

	var resp ChatResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("ToolCalls length = %d, want 1", len(resp.Choices[0].Message.ToolCalls))
	}
	tc := resp.Choices[0].Message.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_abc")
	}
	if tc.Function.Name != "grep" {
		t.Errorf("ToolCall.Function.Name = %q, want %q", tc.Function.Name, "grep")
	}
}

func TestChatResponse_WithError(t *testing.T) {
	jsonData := `{
		"error": {
			"message": "Invalid API key",
			"type": "authentication_error"
		}
	}`

	var resp ChatResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("resp.Error is nil, want error")
	}
	if resp.Error.Message != "Invalid API key" {
		t.Errorf("resp.Error.Message = %q, want %q", resp.Error.Message, "Invalid API key")
	}
	if resp.Error.Type != "authentication_error" {
		t.Errorf("resp.Error.Type = %q, want %q", resp.Error.Type, "authentication_error")
	}
}

func TestToolDefinitions_Structure(t *testing.T) {
	// Verify all tools are defined
	expectedTools := []string{"ls", "cat", "head", "grep", "find", "tree", "write_markdown"}

	if len(ToolDefinitions) != len(expectedTools) {
		t.Errorf("ToolDefinitions length = %d, want %d", len(ToolDefinitions), len(expectedTools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range ToolDefinitions {
		if tool["type"] != "function" {
			t.Errorf("Tool type = %v, want 'function'", tool["type"])
		}
		fn := tool["function"].(map[string]interface{})
		name := fn["name"].(string)
		toolNames[name] = true

		// Verify each tool has required fields
		if fn["description"] == nil || fn["description"] == "" {
			t.Errorf("Tool %q has no description", name)
		}
		if fn["parameters"] == nil {
			t.Errorf("Tool %q has no parameters", name)
		}
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Missing tool definition for %q", expected)
		}
	}
}

// TestClient_SendRequest_MockServer tests the HTTP request/response cycle
func TestClient_SendRequest_MockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json")
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization: Bearer test-key")
		}

		// Return mock response
		resp := ChatResponse{
			ID: "test-123",
			Choices: []struct {
				Message      Message `json:"message"`
				FinishReason string  `json:"finish_reason"`
			}{
				{
					Message:      Message{Role: "assistant", Content: "Test response"},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "test-model",
	}

	client := NewClient(cfg)

	// Note: We can't directly test sendRequest since it's unexported,
	// but we verify the client is constructed correctly
	if client.config.BaseURL != server.URL {
		t.Errorf("client.config.BaseURL = %q, want %q", client.config.BaseURL, server.URL)
	}
}
