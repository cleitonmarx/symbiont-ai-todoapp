package skillregistry

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestBuildSelectionInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messages    []domain.AssistantMessage
		maxChars    int
		recentLimit int
		wantCurrent string
		wantRecent  string
	}{
		{
			name: "current-and-recent-user-inputs",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "plan trip to tokyo"},
				{Role: domain.ChatRole_Assistant, Content: "What dates?"},
				{Role: domain.ChatRole_User, Content: "april 5 to 18"},
			},
			maxChars:    400,
			recentLimit: 3,
			wantCurrent: "plan trip to tokyo\nWhat dates?\napril 5 to 18",
			wantRecent:  "plan trip to tokyo",
		},
		{
			name: "respects-recent-limit",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "u1"},
				{Role: domain.ChatRole_User, Content: "u2"},
				{Role: domain.ChatRole_User, Content: "u3"},
				{Role: domain.ChatRole_User, Content: "u4"},
			},
			maxChars:    400,
			recentLimit: 2,
			wantCurrent: "u4",
			wantRecent:  "u2\nu3",
		},
		{
			name: "short-follow-up-with-location-inherits-assistant-question",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "build a plan for an in-person conference trip"},
				{Role: domain.ChatRole_Assistant, Content: "Which city is it in?"},
				{Role: domain.ChatRole_User, Content: "Berlin."},
			},
			maxChars:    400,
			recentLimit: 3,
			wantCurrent: "build a plan for an in-person conference trip\nWhich city is it in?\nBerlin.",
			wantRecent:  "build a plan for an in-person conference trip",
		},
		{
			name: "returns-empty-without-user-message",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_System, Content: "system"},
			},
			maxChars:    400,
			recentLimit: 3,
			wantCurrent: "",
			wantRecent:  "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotCurrent, gotRecent := buildSelectionInputs(tt.messages, tt.maxChars, tt.recentLimit)
			assert.Equal(t, tt.wantCurrent, gotCurrent)
			assert.Equal(t, tt.wantRecent, gotRecent)
		})
	}
}
