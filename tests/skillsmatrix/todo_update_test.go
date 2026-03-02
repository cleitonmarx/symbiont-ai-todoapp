//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_TodoUpdate(t *testing.T) {
	t.Parallel()

	registry := newSkillMatrixRegistry(t)
	tests := []skillMatrixCase{
		{
			name: "mark-done",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: `Mark my todo "Integration Test Todo" as done.`},
			},
			wantTop:     "todo-update",
			wantContain: []string{"todo-update"},
		},
		{
			name: "statement-implies-update",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "My dentist todo is done."},
			},
			wantTop:     "todo-update",
			wantContain: []string{"todo-update"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
