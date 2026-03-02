//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_TodoSummary(t *testing.T) {
	t.Parallel()

	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"summary-by-topic": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Give me a concise summary of my medical appointments."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		"summarize-topical-items": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Summarize my medical appointments."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		"count-topical-items": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "How many medical appointments do I have?"},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		"summarize-related-tasks": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Give me a concise summary of tasks related to taxes."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		"find-and-summarize": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Find my matching todos and summarize them."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
