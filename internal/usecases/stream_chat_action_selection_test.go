package usecases

import (
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildToolSelectionText(t *testing.T) {
	tests := map[string]struct {
		messages          []domain.AssistantMessage
		expectedText      string
		expectedLen       int
		expectedHasSuffix string
	}{
		"uses-current-message-for-non-ambiguous-input": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "show overdue todos"},
				{Role: domain.ChatRole_Assistant, Content: "Sure"},
				{Role: domain.ChatRole_User, Content: "mark task 123 as done"},
			},
			expectedText: "mark task 123 as done",
		},
		"includes-previous-user-message-for-ambiguous-input": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "mark the grocery task as done"},
				{Role: domain.ChatRole_Assistant, Content: "Done"},
				{Role: domain.ChatRole_User, Content: "do it again"},
			},
			expectedText: "mark the grocery task as done\ndo it again",
		},
		"truncates-to-last-max-chars": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: strings.Repeat("a", MAX_TOOL_SELECTION_CHARS)},
				{Role: domain.ChatRole_User, Content: "do it"},
			},
			expectedLen:       MAX_TOOL_SELECTION_CHARS,
			expectedHasSuffix: "do it",
		},
		"returns-empty-when-no-user-message-exists": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_Assistant, Content: "No user content"},
			},
			expectedText: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			selectionText := buildActionSelectionText(tt.messages)

			if tt.expectedText != "" || (tt.expectedLen == 0 && tt.expectedHasSuffix == "") {
				assert.Equal(t, tt.expectedText, selectionText)
			}
			if tt.expectedLen > 0 {
				require.Len(t, []rune(selectionText), tt.expectedLen)
			}
			if tt.expectedHasSuffix != "" {
				assert.True(t, strings.HasSuffix(selectionText, tt.expectedHasSuffix))
			}
		})
	}
}
