package usecases

import (
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestBuildToolSelectionText(t *testing.T) {
	t.Parallel()

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
				{Role: domain.ChatRole_User, Content: strings.Repeat("a", MAX_SKILLS_SELECTION_CHARS)},
				{Role: domain.ChatRole_User, Content: "do it"},
			},
			expectedLen:       MAX_SKILLS_SELECTION_CHARS,
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
				assert.Len(t, []rune(selectionText), tt.expectedLen)
			}
			if tt.expectedHasSuffix != "" {
				assert.True(t, strings.HasSuffix(selectionText, tt.expectedHasSuffix))
			}
		})
	}
}

func TestBuildSkillsPrompt(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		skills       []domain.AssistantSkillDefinition
		expectEmpty  bool
		expectedText []string
		notContains  []string
		maxLen       int
	}{
		"returns-empty-when-no-skills": {
			skills:      nil,
			expectEmpty: true,
		},
		"includes-skill-content-and-tools": {
			skills: []domain.AssistantSkillDefinition{
				{
					Name:      "todo-mutation-safety",
					UseWhen:   "User asks to update todos",
					AvoidWhen: "User only wants summaries",
					Tools:     []string{"update_todos", "fetch_todos", "update_todos"},
					Content:   "1. Fetch ids first\n2. Confirm intent\n3. Mutate",
				},
			},
			expectedText: []string{
				"Skill runbooks for this turn:",
				"Skill: todo-mutation-safety",
				"Use when: User asks to update todos",
				"Avoid when: User only wants summaries",
				"Tools: update_todos, fetch_todos",
				"Workflow:",
				"1. Fetch ids first",
			},
		},
		"ignores-empty-skill-name": {
			skills: []domain.AssistantSkillDefinition{
				{Name: "   ", Content: "ignore me"},
			},
			expectedText: []string{
				"Skill runbooks for this turn:",
			},
			notContains: []string{
				"Skill:",
				"ignore me",
			},
		},
		"truncates-long-prompt": {
			skills: []domain.AssistantSkillDefinition{
				{
					Name:    "verbose_skill",
					UseWhen: strings.Repeat("x", MAX_SKILLS_PROMPT_CHARS),
				},
			},
			maxLen: MAX_SKILLS_PROMPT_CHARS,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := buildSkillsPrompt(tt.skills)

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
				assert.LessOrEqual(t, len([]rune(got)), tt.maxLen)
			}
		})
	}
}
