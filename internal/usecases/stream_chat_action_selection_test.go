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
				{Role: domain.ChatRole_User, Content: strings.Repeat("a", MAX_ACTION_SELECTION_CHARS)},
				{Role: domain.ChatRole_User, Content: "do it"},
			},
			expectedLen:       MAX_ACTION_SELECTION_CHARS,
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

func TestBuildToolingPrompt(t *testing.T) {
	tests := map[string]struct {
		actions      []domain.AssistantActionDefinition
		expectEmpty  bool
		expectedText []string
		notContains  []string
		maxLen       int
	}{
		"returns-empty-when-no-actions": {
			actions:     nil,
			expectEmpty: true,
		},
		"includes-hints-for-relevant-actions": {
			actions: []domain.AssistantActionDefinition{
				{
					Name: "set_ui_filters",
					Hints: domain.AssistantActionHints{
						UseWhen:   "read intents",
						AvoidWhen: "writes",
						ArgRules:  "allowed keys only",
					},
				},
				{
					Name: "fetch_todos",
					Hints: domain.AssistantActionHints{
						UseWhen:  "disambiguation",
						ArgRules: "page and page_size required",
					},
				},
				{
					Name: "create_todo",
					Hints: domain.AssistantActionHints{
						UseWhen:  "create todos",
						ArgRules: "title and due_date required",
					},
				},
			},
			expectedText: []string{
				"Tooling rules for this turn:",
				"fetch_todos",
				"create_todo",
				"set_ui_filters",
				"Use: read intents",
				"Avoid: writes",
				"Args: allowed keys only",
				"fetch_todos",
			},
		},
		"uses-fallback-when-hints-are-empty": {
			actions: []domain.AssistantActionDefinition{
				{Name: "unknown_action"},
			},
			expectedText: []string{
				"unknown_action",
				"Follow the tool schema and description.",
			},
		},
		"limits-tools-in-scope": {
			actions: []domain.AssistantActionDefinition{
				{Name: "a", Hints: domain.AssistantActionHints{UseWhen: "u"}},
				{Name: "b", Hints: domain.AssistantActionHints{UseWhen: "u"}},
				{Name: "c", Hints: domain.AssistantActionHints{UseWhen: "u"}},
				{Name: "d", Hints: domain.AssistantActionHints{UseWhen: "u"}},
			},
			expectedText: []string{"- a:", "- b:", "- c:"},
			notContains:  []string{"- d:"},
		},
		"truncates-long-prompt": {
			actions: []domain.AssistantActionDefinition{
				{
					Name: "verbose_tool",
					Hints: domain.AssistantActionHints{
						UseWhen: strings.Repeat("x", MAX_ACTION_PROMPT_CHARS),
					},
				},
			},
			maxLen: MAX_ACTION_PROMPT_CHARS,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := buildActionsPrompt(tt.actions)

			if tt.expectEmpty {
				assert.Equal(t, "", got)
				return
			}

			for _, expect := range tt.expectedText {
				assert.Contains(t, got, expect)
			}
			for _, denied := range tt.notContains {
				assert.NotContains(t, got, denied)
			}
			if tt.maxLen > 0 {
				require.LessOrEqual(t, len([]rune(got)), tt.maxLen)
			}
		})
	}
}
