//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_Boundaries(t *testing.T) {
	t.Parallel()

	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"show-topical-items": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Show my medical appointments."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
		"summarize-topical-items": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Summarize my medical appointments."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		"show-done-topical-items": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Show done dentist todos."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
		"statement-implies-update": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "My dentist todo is done."},
			},
			wantTop:     "todo-update",
			wantContain: []string{"todo-update"},
		},
		"single-concrete-create": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Create a todo to renew my passport tomorrow."},
			},
			wantTop:     "todo-create",
			wantContain: []string{"todo-create"},
		},
		"full-plan-before-deadline": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Create a full plan to renew my passport before May."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"research-only": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Research the current requirements online."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"research-and-create-plan": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Research the current requirements online and create a plan."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
