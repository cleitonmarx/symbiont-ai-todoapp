//go:build skillmatrix

package skillsmatrix

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
)

func TestSkillRelevancePromptMatrix_WebResearch(t *testing.T) {
	registry := newSkillMatrixRegistry(t)
	tests := map[string]skillMatrixCase{
		"web-research": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Search the web for the current requirements and summarize the key points."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"research-only": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Research the current requirements online."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"open-external-website-and-read-title": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Open the external website https://duckduckgo.com/ and tell me only the page title."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"web-research-after-trip-plan-follow-up": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: `Plan a trip to Tokyo from April 4-14. Research for the best hotels and ramen restaurants, and create a complete todo plan for it. Use task prefix: "Japan Trip:"`},
				{Role: assistant.ChatRole_Assistant, Content: `I created a complete todo plan for your Tokyo trip from April 4-14 with the prefix "Japan Trip:". It includes tasks for booking hotels, researching and visiting top ramen spots, preparing travel documents, packing, and scheduling daily activities.`},
				{Role: assistant.ChatRole_User, Content: "What about my hotel's research? List the top 3."},
				{Role: assistant.ChatRole_Assistant, Content: `You have 2 open hotel-related tasks: "Research and book hotel in Tokyo" and "Confirm hotel reservation and transportation in Tokyo".`},
				{Role: assistant.ChatRole_User, Content: "Research the web for the top 3 hotels in Tokyo."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"test001": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Research the internet for grocery stores near me in Maple Ridge, BC."},
			},
			wantTop:     "web-research",
			wantContain: []string{"web-research"},
		},
		"web-research-after-summary-and-view-context-shift": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "Make a concise summary of my open todos from June."},
				{Role: assistant.ChatRole_Assistant, Content: "You have 35 open todos for June, spanning personal tasks and work-related planning."},
				{Role: assistant.ChatRole_User, Content: "Show it on my view."},
				{Role: assistant.ChatRole_Assistant, Content: "Your view is now filtered to show all open todos due in June."},
				{Role: assistant.ChatRole_User, Content: "Research on the web the events happening in Maple Ridge in June."},
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
