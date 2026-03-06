package mcp

import (
	"context"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
)

// mcpToolAction adapts one discovered MCP tool to the Action contract.
type mcpToolAction struct {
	definition    assistant.ActionDefinition
	statusMessage string
	renderer      assistant.ActionResultRenderer
	execute       func(context.Context, assistant.ActionCall, []assistant.Message) assistant.Message
}

// Definition returns the static action definition associated with this MCP tool.
func (a mcpToolAction) Definition() assistant.ActionDefinition {
	return a.definition
}

// StatusMessage returns a per-tool execution status string for UI streaming updates.
func (a mcpToolAction) StatusMessage() string {
	if msg := strings.TrimSpace(a.statusMessage); msg != "" {
		return msg
	}

	name := strings.TrimSpace(a.definition.Name)
	if name == "" {
		return defaultStatusMessage
	}
	return "⏳ Running " + name + "..."
}

// Renderer returns the deterministic renderer configured for this MCP tool, when available.
func (a mcpToolAction) Renderer() (assistant.ActionResultRenderer, bool) {
	if a.renderer == nil {
		return nil, false
	}
	return a.renderer, true
}

// Execute delegates execution to the registry callback bound at initialization time.
func (a mcpToolAction) Execute(ctx context.Context, call assistant.ActionCall, history []assistant.Message) assistant.Message {
	if a.execute == nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: common.Ptr(call.ID),
			Content:      "errors[1]{error,details}mcp_call_error,action is not executable",
		}
	}
	return a.execute(ctx, call, history)
}
