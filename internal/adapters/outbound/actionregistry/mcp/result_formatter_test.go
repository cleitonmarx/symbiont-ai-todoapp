package mcp

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteCodeFormatter_Format(t *testing.T) {
	t.Parallel()

	formatter := executeCodeFormatter{}
	tests := map[string]struct {
		actionResult string
		assert       func(*testing.T, domain.AssistantMessage)
	}{
		"formats-result": {
			actionResult: `{"result":["line1","line2"]}`,
			assert: func(t *testing.T, msg domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, msg.Role)
				require.NotNil(t, msg.ActionCallID)
				assert.Equal(t, "line1\nline2", msg.Content)
			},
		},
		"formats-errors": {
			actionResult: `{"error":["boom"]}`,
			assert: func(t *testing.T, msg domain.AssistantMessage) {
				assert.Contains(t, msg.Content, "code_error")
				assert.Contains(t, msg.Content, "boom")
			},
		},
		"invalid-json-produces-empty-content": {
			actionResult: `not-json`,
			assert: func(t *testing.T, msg domain.AssistantMessage) {
				assert.Equal(t, "", msg.Content)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			msg := formatter.Format(tt.actionResult, domain.AssistantActionCall{ID: "call-1"})
			require.NotNil(t, msg.ActionCallID)
			assert.Equal(t, "call-1", *msg.ActionCallID)
			tt.assert(t, msg)
		})
	}
}

func TestResultFormatters(t *testing.T) {
	t.Parallel()

	formatter, found := resultFormatters["execute_code"]
	require.True(t, found)
	require.NotNil(t, formatter)
}
