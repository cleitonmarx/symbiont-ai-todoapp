//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

func TestSkillRelevancePromptMatrix_WebResearch(t *testing.T) {
	t.Parallel()

	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"web-research": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Search the web for the current requirements and summarize the key points."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"research-only": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Research the current requirements online."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"open-external-website-and-read-title": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Open the external website https://duckduckgo.com/ and tell me only the page title."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"web-research-after-trip-plan-follow-up": {
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: `Plan a trip to Tokyo from April 4-14. Research for the best hotels and ramen restaurants, and create a complete todo plan for it. Use task prefix: "Japan Trip:"`},
				{Role: domain.ChatRole_Assistant, Content: `I created a complete todo plan for your Tokyo trip from April 4-14 with the prefix "Japan Trip:". It includes tasks for booking hotels, researching and visiting top ramen spots, preparing travel documents, packing, and scheduling daily activities.`},
				{Role: domain.ChatRole_User, Content: "What about my hotel's research? List the top 3."},
				{Role: domain.ChatRole_Assistant, Content: `You have 2 open hotel-related tasks: "Research and book hotel in Tokyo" and "Confirm hotel reservation and transportation in Tokyo".`},
				{Role: domain.ChatRole_User, Content: "Research the web for the top 3 hotels in Tokyo."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runSkillMatrixCase(t, registry, tc)
		})
	}
}
