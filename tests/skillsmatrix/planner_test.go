//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
)

func TestSkillRelevancePromptMatrix_TodoGoalPlanner(t *testing.T) {

	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"goal-planner": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Plan a trip to Tokyo in April and create a todo plan for it."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"full-plan-before-deadline": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Create a full plan to renew my passport before May."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"recent-context-helps-persistence": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Plan a trip to Tokyo in April."},
				{Role: assistant.ChatRole_Assistant, Content: "What dates are you considering?"},
				{Role: assistant.ChatRole_User, Content: "April 9 to 30."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"continuation-with-budget-only": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Create a full plan for moving apartment next month."},
				{Role: assistant.ChatRole_Assistant, Content: "What budget should I assume?"},
				{Role: assistant.ChatRole_User, Content: "$1500."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"continuation-with-location-only": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Build a plan for an in-person conference trip."},
				{Role: assistant.ChatRole_Assistant, Content: "Which city is it in?"},
				{Role: assistant.ChatRole_User, Content: "Berlin."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"continuation-with-deadline-only": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Create an end-to-end study plan for my Go interview."},
				{Role: assistant.ChatRole_Assistant, Content: "When is the interview?"},
				{Role: assistant.ChatRole_User, Content: "May 12."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"continuation-with-noisy-turns-within-limit": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Create an end-to-end relocation plan for next quarter."},
				{Role: assistant.ChatRole_Assistant, Content: "What budget should I assume?"},
				{Role: assistant.ChatRole_User, Content: "yes"},
				{Role: assistant.ChatRole_Assistant, Content: "Do you have a hard deadline?"},
				{Role: assistant.ChatRole_User, Content: "no"},
				{Role: assistant.ChatRole_Assistant, Content: "What location should I use?"},
				{Role: assistant.ChatRole_User, Content: "Lisbon."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"continuation-at-recent-limit": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Create an end-to-end relocation plan for next quarter."},
				{Role: assistant.ChatRole_Assistant, Content: "What budget should I assume?"},
				{Role: assistant.ChatRole_User, Content: "yes"},
				{Role: assistant.ChatRole_Assistant, Content: "Do you have a hard deadline?"},
				{Role: assistant.ChatRole_User, Content: "no"},
				{Role: assistant.ChatRole_Assistant, Content: "How strict is the timeline?"},
				{Role: assistant.ChatRole_User, Content: "flexible"},
				{Role: assistant.ChatRole_Assistant, Content: "What location should I use?"},
				{Role: assistant.ChatRole_User, Content: "Lisbon."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"continuation-beyond-recent-limit": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Create an end-to-end relocation plan for next quarter."},
				{Role: assistant.ChatRole_Assistant, Content: "What budget should I assume?"},
				{Role: assistant.ChatRole_User, Content: "yes"},
				{Role: assistant.ChatRole_Assistant, Content: "Do you have a hard deadline?"},
				{Role: assistant.ChatRole_User, Content: "no"},
				{Role: assistant.ChatRole_Assistant, Content: "How strict is the timeline?"},
				{Role: assistant.ChatRole_User, Content: "flexible"},
				{Role: assistant.ChatRole_Assistant, Content: "Any other preference I should carry?"},
				{Role: assistant.ChatRole_User, Content: "fine"},
				{Role: assistant.ChatRole_Assistant, Content: "What location should I use?"},
				{Role: assistant.ChatRole_User, Content: "Lisbon."},
			},
			wantTop: "",
		},
		"research-and-create-plan": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Research Japan visa requirements and create a plan."},
			},
			wantTop:     "todo-goal-planner",
			wantContain: []string{"todo-goal-planner"},
		},
		"research-and-create-tasks": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Research the requirements and create tasks for me."},
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
