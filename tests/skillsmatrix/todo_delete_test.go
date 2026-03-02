//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_TodoDelete(t *testing.T) {
	t.Parallel()

	registry := newSkillMatrixRegistry(t)
	tests := []skillMatrixCase{
		{
			name: "delete-by-title",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: `Delete my todo "Integration Test Todo".`},
			},
			wantTop:     "todo-delete",
			wantContain: []string{"todo-delete"},
		},
		{
			name: "delete-fuzzy-set",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Delete old job application todos."},
			},
			wantTop:     "todo-delete",
			wantContain: []string{"todo-delete"},
		},
		{
			name: "delete-completed-set",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Remove completed todos from my list."},
			},
			wantTop:     "todo-delete",
			wantContain: []string{"todo-delete"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
