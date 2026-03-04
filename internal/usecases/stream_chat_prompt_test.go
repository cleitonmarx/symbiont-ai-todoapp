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

func TestCompactToLastMessages(t *testing.T) {
	t.Parallel()

	msgs := []domain.AssistantMessage{
		{Role: domain.ChatRole_User, Content: "one"},
		{Role: domain.ChatRole_Assistant, Content: "two"},
		{Role: domain.ChatRole_User, Content: "three"},
	}

	tests := map[string]struct {
		messages  []domain.AssistantMessage
		max       int
		wantTexts []string
	}{
		"returns-nil-when-max-non-positive": {
			messages: msgs,
			max:      0,
		},
		"returns-nil-when-empty": {
			max: 2,
		},
		"returns-copy-when-within-limit": {
			messages:  msgs[:2],
			max:       3,
			wantTexts: []string{"one", "two"},
		},
		"returns-last-messages-when-over-limit": {
			messages:  msgs,
			max:       2,
			wantTexts: []string{"two", "three"},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := compactToLastMessages(tt.messages, tt.max)
			assert.Len(t, got, len(tt.wantTexts))
			for i, want := range tt.wantTexts {
				assert.Equal(t, want, got[i].Content)
			}
			if len(got) > 0 && len(tt.messages) > 0 {
				assert.NotSame(t, &tt.messages[0], &got[0])
			}
		})
	}
}

func TestTruncateToFirstChars(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		max   int
		want  string
	}{
		"returns-empty-when-max-non-positive": {
			input: "hello",
			max:   0,
			want:  "",
		},
		"trims-and-keeps-when-short-enough": {
			input: "  hello  ",
			max:   10,
			want:  "hello",
		},
		"truncates-by-rune": {
			input: "ábcdef",
			max:   3,
			want:  "ábc",
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, truncateToFirstChars(tt.input, tt.max))
		})
	}
}
