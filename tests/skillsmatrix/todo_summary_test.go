//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_TodoSummary(t *testing.T) {
	t.Parallel()

	registry := newSkillMatrixRegistry(t)
	tests := []skillMatrixCase{
		{
			name: "summary-by-topic",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Give me a concise summary of my medical appointments."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		{
			name: "summarize-topical-items",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Summarize my medical appointments."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		{
			name: "count-topical-items",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "How many medical appointments do I have?"},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		{
			name: "summarize-related-tasks",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Give me a concise summary of tasks related to taxes."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		{
			name: "find-and-summarize",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Find my Tokyo trip todos and summarize them."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
