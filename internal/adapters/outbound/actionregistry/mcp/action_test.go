package mcp

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPToolAction_Methods(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		action       mcpToolAction
		assertResult func(*testing.T, mcpToolAction)
	}{
		"definition": {
			action: mcpToolAction{definition: domain.AssistantActionDefinition{Name: "search"}},
			assertResult: func(t *testing.T, action mcpToolAction) {
				assert.Equal(t, "search", action.Definition().Name)
			},
		},
		"status-message-custom": {
			action: mcpToolAction{statusMessage: "custom"},
			assertResult: func(t *testing.T, action mcpToolAction) {
				assert.Equal(t, "custom", action.StatusMessage())
			},
		},
		"status-message-by-name": {
			action: mcpToolAction{definition: domain.AssistantActionDefinition{Name: "fetch"}},
			assertResult: func(t *testing.T, action mcpToolAction) {
				assert.Equal(t, "⏳ Running fetch...", action.StatusMessage())
			},
		},
		"status-message-default": {
			action: mcpToolAction{},
			assertResult: func(t *testing.T, action mcpToolAction) {
				assert.Equal(t, defaultStatusMessage, action.StatusMessage())
			},
		},
		"renderer-present": {
			action: mcpToolAction{renderer: fakeRenderer{ok: true}},
			assertResult: func(t *testing.T, action mcpToolAction) {
				renderer, ok := action.Renderer()
				require.True(t, ok)
				require.NotNil(t, renderer)
			},
		},
		"renderer-missing": {
			action: mcpToolAction{},
			assertResult: func(t *testing.T, action mcpToolAction) {
				renderer, ok := action.Renderer()
				assert.False(t, ok)
				assert.Nil(t, renderer)
			},
		},
		"execute-delegates": {
			action: mcpToolAction{execute: func(_ context.Context, call domain.AssistantActionCall, _ []domain.AssistantMessage) domain.AssistantMessage {
				return domain.AssistantMessage{Role: domain.ChatRole_Tool, Content: call.Name}
			}},
			assertResult: func(t *testing.T, action mcpToolAction) {
				msg := action.Execute(context.Background(), domain.AssistantActionCall{Name: "fetch"}, nil)
				assert.Equal(t, domain.ChatRole_Tool, msg.Role)
				assert.Equal(t, "fetch", msg.Content)
			},
		},
		"execute-default-error": {
			action: mcpToolAction{},
			assertResult: func(t *testing.T, action mcpToolAction) {
				msg := action.Execute(context.Background(), domain.AssistantActionCall{ID: "call-1"}, nil)
				assert.Equal(t, domain.ChatRole_Tool, msg.Role)
				require.NotNil(t, msg.ActionCallID)
				assert.Equal(t, "call-1", *msg.ActionCallID)
				assert.Contains(t, msg.Content, "mcp_call_error")
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tt.assertResult(t, tt.action)
		})
	}
}
