package mcp

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
)

func TestExecuteCodeToolFormatter_FormatResult(t *testing.T) {
	t.Parallel()

	formatter := executeCodeToolFormatter{}
	tests := map[string]struct {
		actionResult string
		assert       func(*testing.T, assistant.Message)
	}{
		"formats-result": {
			actionResult: `{"result":["line1","line2"]}`,
			assert: func(t *testing.T, msg assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, msg.Role)
				assert.NotNil(t, msg.ActionCallID)
				assert.Equal(t, "line1\nline2", msg.Content)
			},
		},
		"formats-errors": {
			actionResult: `{"error":["boom"]}`,
			assert: func(t *testing.T, msg assistant.Message) {
				assert.Contains(t, msg.Content, "code_error")
				assert.Contains(t, msg.Content, "boom")
			},
		},
		"invalid-json-produces-empty-content": {
			actionResult: `not-json`,
			assert: func(t *testing.T, msg assistant.Message) {
				assert.Equal(t, "", msg.Content)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			msg := formatter.FormatResult(tt.actionResult, assistant.ActionCall{ID: "call-1"})
			assert.NotNil(t, msg.ActionCallID)
			assert.Equal(t, "call-1", *msg.ActionCallID)
			tt.assert(t, msg)
		})
	}
}

func TestExecuteCodeToolFormatter_FormatArguments(t *testing.T) {
	t.Parallel()

	formatter := executeCodeToolFormatter{}
	tests := map[string]struct {
		input map[string]any
		want  map[string]any
	}{
		"normalizes-escaped-newlines": {
			input: map[string]any{"code": `result = 1\nresult`},
			want:  map[string]any{"code": "result = 1\nresult"},
		},
		"leaves-multiline-code-unchanged": {
			input: map[string]any{"code": "result = 1\nresult"},
			want:  map[string]any{"code": "result = 1\nresult"},
		},
		"ignores-missing-code": {
			input: map[string]any{"session_id": 123},
			want:  map[string]any{"session_id": 123},
		},
	}

	for name, tt := range tests {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := formatter.FormatArguments(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToolFormatters(t *testing.T) {
	t.Parallel()

	formatter, found := toolFormatters["execute_code"]
	assert.True(t, found)
	assert.NotNil(t, formatter)
}

func TestToolFormatterRegistry_FormatResult(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		toolName     string
		actionResult string
		wantFound    bool
		assert       func(*testing.T, assistant.Message)
	}{
		"formats-known-tool": {
			toolName:     "execute_code",
			actionResult: `{"result":["ok"]}`,
			wantFound:    true,
			assert: func(t *testing.T, msg assistant.Message) {
				assert.NotNil(t, msg.ActionCallID)
				assert.Equal(t, "ok", msg.Content)
			},
		},
		"returns-not-found-for-unknown-tool": {
			toolName:     "missing",
			actionResult: "{}",
			wantFound:    false,
			assert: func(t *testing.T, msg assistant.Message) {
				assert.Equal(t, assistant.Message{}, msg)
			},
		},
	}

	for name, tt := range tests {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			msg, found := toolFormatters.FormatResult(tt.toolName, tt.actionResult, assistant.ActionCall{ID: "call-1"})
			assert.Equal(t, tt.wantFound, found)
			tt.assert(t, msg)
		})
	}
}

func TestToolFormatterRegistry_FormatArguments(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		toolName  string
		input     map[string]any
		want      map[string]any
		wantFound bool
	}{
		"formats-known-tool": {
			toolName:  "execute_code",
			input:     map[string]any{"code": `result = 1\nresult`},
			want:      map[string]any{"code": "result = 1\nresult"},
			wantFound: true,
		},
		"returns-not-found-for-unknown-tool": {
			toolName:  "missing",
			input:     map[string]any{"code": "x"},
			want:      map[string]any{"code": "x"},
			wantFound: false,
		},
	}

	for name, tt := range tests {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			args, found := toolFormatters.FormatArguments(tt.toolName, tt.input)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.want, args)
		})
	}
}
