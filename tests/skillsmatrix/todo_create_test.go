//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_TodoCreate(t *testing.T) {
	t.Parallel()

	registry := newSkillMatrixRegistry(t)
	tests := []skillMatrixCase{
		{
			name: "create-todo",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: `Create a todo called "Renew passport" due tomorrow.`},
			},
			wantTop:     "todo-create",
			wantContain: []string{"todo-create"},
		},
		{
			name: "single-concrete-create",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Create a todo to renew my passport tomorrow."},
			},
			wantTop:     "todo-create",
			wantContain: []string{"todo-create"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
