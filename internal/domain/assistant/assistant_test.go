package assistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSelectedSkill(t *testing.T) {
	t.Parallel()

	definition := SkillDefinition{
		Name:   "update_todos",
		Source: "skills/update_todos.md",
		Tools:  []string{"fetch_todos", "update_todos"},
	}

	got := NewSelectedSkill(definition)

	assert.Equal(t, SelectedSkill{
		Name:   "update_todos",
		Source: "skills/update_todos.md",
		Tools:  []string{"fetch_todos", "update_todos"},
	}, got)

	definition.Tools[0] = "mutated"
	assert.Equal(t, []string{"fetch_todos", "update_todos"}, got.Tools)
}

func TestActionDefinition_RequiresApproval(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		definition ActionDefinition
		want       bool
	}{
		"required": {
			definition: ActionDefinition{
				Approval: ActionApproval{
					Required: true,
				},
			},
			want: true,
		},
		"not-required": {
			definition: ActionDefinition{
				Approval: ActionApproval{
					Required: false,
				},
			},
			want: false,
		},
		"zero-value": {
			definition: ActionDefinition{},
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
