package mcp

import (
	"context"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// mcpToolAction adapts one discovered MCP tool to the AssistantAction contract.
type mcpToolAction struct {
	definition    domain.AssistantActionDefinition
	statusMessage string
	renderer      domain.ActionResultRenderer
	execute       func(context.Context, domain.AssistantActionCall, []domain.AssistantMessage) domain.AssistantMessage
}

// Definition returns the static action definition associated with this MCP tool.
func (a mcpToolAction) Definition() domain.AssistantActionDefinition {
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
func (a mcpToolAction) Renderer() (domain.ActionResultRenderer, bool) {
	if a.renderer == nil {
		return nil, false
	}
	return a.renderer, true
}

// Execute delegates execution to the registry callback bound at initialization time.
func (a mcpToolAction) Execute(ctx context.Context, call domain.AssistantActionCall, history []domain.AssistantMessage) domain.AssistantMessage {
	if a.execute == nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: common.Ptr(call.ID),
			Content:      "errors[1]{error,details}mcp_call_error,action is not executable",
		}
	}
	return a.execute(ctx, call, history)
}
