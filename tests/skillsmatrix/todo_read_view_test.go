//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_TodoReadView(t *testing.T) {
	t.Parallel()

	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"read-with-date-range": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "List my open todos due from March 1-7."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
		"show-topical-items": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Show my medical appointments."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
		"show-done-topical-items": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Show done dentist todos."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
		"between-date-phrasing": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "List my open todos between March 1 and March 7."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
		"relative-week-phrasing": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "List my open todos due this week."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
		"relative-next-month-phrasing": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "List my todos due next month."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
		"find-related-tasks": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Find todos related to taxes."},
			},
			wantTop:     "todo-read-view",
			wantContain: []string{"todo-read-view"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
