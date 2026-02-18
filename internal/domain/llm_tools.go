package domain

import "context"

// LLMTool represents a tool that can be executed by the LLM.
type LLMTool interface {
	// Definition returns the LLMTool definition.
	Definition() LLMToolDefinition
	// StatusMessage returns a user-friendly status line for this tool.
	StatusMessage() string
	// Call executes the tool with the given tool call and chat messages.
	Call(context.Context, LLMStreamEventToolCall, []LLMChatMessage) LLMChatMessage
}

// LLMToolRegistry defines the interface for calling registered LLM tools.
type LLMToolRegistry interface {
	// Call executes the tool with the given tool call and chat messages.
	Call(context.Context, LLMStreamEventToolCall, []LLMChatMessage) LLMChatMessage
	// StatusMessage returns a friendly status message for the given tool name.
	StatusMessage(toolName string) string
	// List returns all registered LLM tools.
	List() []LLMToolDefinition
}

// LLMToolDefinition represents a tool that can be used by the LLM.
type LLMToolDefinition struct {
	Type     string
	Function LLMToolFunction
}

// LLMToolFunction represents a function tool for the LLM.
type LLMToolFunction struct {
	Description string
	Name        string
	Parameters  LLMToolFunctionParameters
}

// LLMToolFunctionParameters represents the parameters schema for a function tool.
type LLMToolFunctionParameters struct {
	Type       string
	Properties map[string]LLMToolFunctionParameterDetail
}

// LLMToolFunctionParameterDetail represents a single parameter in the function tool schema.
type LLMToolFunctionParameterDetail struct {
	Type        string
	Description string
	Required    bool
}
