//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
)

func TestSkillRelevancePromptMatrix_TodoUpdate(t *testing.T) {

	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"mark-done": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: `Mark my todo "Integration Test Todo" as done.`},
			},
			wantTop:     "todo-update",
			wantContain: []string{"todo-update"},
		},
		"statement-implies-update": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "My dentist todo is done."},
			},
			wantTop:     "todo-update",
			wantContain: []string{"todo-update"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
