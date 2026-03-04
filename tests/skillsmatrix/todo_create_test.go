//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_TodoCreate(t *testing.T) {

	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"create-todo": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: `Create a todo called "Renew passport" due tomorrow.`},
			},
			wantTop:     "todo-create",
			wantContain: []string{"todo-create"},
		},
		"single-concrete-create": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Create a todo to renew my passport tomorrow."},
			},
			wantTop:     "todo-create",
			wantContain: []string{"todo-create"},
		},
		"integration-create-prompt": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: `Create one todo named "Integration Test Todo" due tomorrow.`},
			},
			wantTop:     "todo-create",
			wantContain: []string{"todo-create"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
