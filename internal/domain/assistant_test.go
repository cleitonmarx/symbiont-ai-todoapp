package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAssistantSelectedSkill(t *testing.T) {
	t.Parallel()

	definition := AssistantSkillDefinition{
		Name:   "update_todos",
		Source: "skills/update_todos.md",
		Tools:  []string{"fetch_todos", "update_todos"},
	}

	got := NewAssistantSelectedSkill(definition)

	assert.Equal(t, AssistantSelectedSkill{
		Name:   "update_todos",
		Source: "skills/update_todos.md",
		Tools:  []string{"fetch_todos", "update_todos"},
	}, got)

	definition.Tools[0] = "mutated"
	assert.Equal(t, []string{"fetch_todos", "update_todos"}, got.Tools)
}

func TestAssistantActionDefinition_RequiresApproval(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		definition AssistantActionDefinition
		want       bool
	}{
		"required": {
			definition: AssistantActionDefinition{
				Approval: AssistantActionApproval{
					Required: true,
				},
			},
			want: true,
		},
		"not-required": {
			definition: AssistantActionDefinition{
				Approval: AssistantActionApproval{
					Required: false,
				},
			},
			want: false,
		},
		"zero-value": {
			definition: AssistantActionDefinition{},
			want:       false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.definition.RequiresApproval()
			assert.Equal(t, tt.want, got)
		})
	}
}
