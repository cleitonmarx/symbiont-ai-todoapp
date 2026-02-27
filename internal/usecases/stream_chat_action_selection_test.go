package usecases

import (
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
)

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
