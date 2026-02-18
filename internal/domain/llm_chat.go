package domain

import "strings"

// LLMChatMessage represents a message in a chat request to the LLM API.
type LLMChatMessage struct {
	Role       ChatRole
	Content    string
	ToolCallID *string
	ToolCalls  []LLMStreamEventToolCall
}

// IsToolCallSuccess returns true if the chat message is a tool call
// and indicates success based on its content.
func (m LLMChatMessage) IsToolCallSuccess() bool {
	return m.Role == ChatRole_Tool &&
		m.ToolCallID != nil &&
		!strings.Contains(m.Content, "error")
}

// LLMChatRequest represents a request to the LLM API.
type LLMChatRequest struct {
	Model    string
	Messages []LLMChatMessage
	Stream   bool
	// Optional parameters.
	Temperature      *float64
	TopP             *float64
	MaxTokens        *int
	FrequencyPenalty *float64
	Tools            []LLMToolDefinition
}

// LLMChatResponse represents the response from a chat request to the LLM API.
type LLMChatResponse struct {
	Content string
	Usage   LLMUsage
}
