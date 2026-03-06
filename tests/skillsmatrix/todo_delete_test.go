//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
)

func TestSkillRelevancePromptMatrix_TodoDelete(t *testing.T) {

	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"delete-by-title": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: `Delete my todo "Integration Test Todo".`},
			},
			wantTop:     "todo-delete",
			wantContain: []string{"todo-delete"},
		},
		"delete-fuzzy-set": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Delete old job application todos."},
			},
			wantTop:     "todo-delete",
			wantContain: []string{"todo-delete"},
		},
		"delete-completed-set": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Remove completed todos from my list."},
			},
			wantTop:     "todo-delete",
			wantContain: []string{"todo-delete"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
