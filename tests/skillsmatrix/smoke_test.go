//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_Smoke(t *testing.T) {
	t.Parallel()

	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"create-todo": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: `Create a todo called "Renew passport" due tomorrow.`},
			},
			wantTop:     "todo-create",
			wantContain: []string{"todo-create"},
		},
		"read-with-date-range": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "List my open todos due from March 1-7."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
		"summary-by-topic": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Give me a concise summary of my medical appointments."},
			},
			wantTop:     "todo-summary",
			wantContain: []string{"todo-summary"},
		},
		"mark-done": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: `Mark my todo "Integration Test Todo" as done.`},
			},
			wantTop:     "todo-update",
			wantContain: []string{"todo-update"},
		},
		"delete-by-title": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: `Delete my todo "Integration Test Todo".`},
			},
			wantTop:     "todo-delete",
			wantContain: []string{"todo-delete"},
		},
		"goal-planner": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Plan a trip to Tokyo in April and create a todo plan for it."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"web-research": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Search the web for the current requirements and summarize the key points."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"casual-hello": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "hello"},
			},
		},
	}

	for name, tc := range tests {

		t.Run(name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
