package skillregistry

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
)

func TestBuildSelectionInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messages    []assistant.Message
		maxChars    int
		recentLimit int
		wantCurrent string
		wantRecent  string
	}{
		{
			name: "current-and-recent-user-inputs",
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "plan trip to tokyo"},
				{Role: assistant.ChatRole_Assistant, Content: "What dates?"},
				{Role: assistant.ChatRole_User, Content: "april 5 to 18"},
			},
			maxChars:    400,
			recentLimit: 3,
			wantCurrent: "plan trip to tokyo\nWhat dates?\napril 5 to 18",
			wantRecent:  "plan trip to tokyo",
		},
		{
			name: "respects-recent-limit",
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "u1"},
				{Role: assistant.ChatRole_User, Content: "u2"},
				{Role: assistant.ChatRole_User, Content: "u3"},
				{Role: assistant.ChatRole_User, Content: "u4"},
			},
			maxChars:    400,
			recentLimit: 2,
			wantCurrent: "u4",
			wantRecent:  "u2\nu3",
		},
		{
			name: "short-follow-up-with-location-inherits-assistant-question",
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "build a plan for an in-person conference trip"},
				{Role: assistant.ChatRole_Assistant, Content: "Which city is it in?"},
				{Role: assistant.ChatRole_User, Content: "Berlin."},
			},
			maxChars:    400,
			recentLimit: 3,
			wantCurrent: "build a plan for an in-person conference trip\nWhich city is it in?\nBerlin.",
			wantRecent:  "build a plan for an in-person conference trip",
		},
		{
			name: "returns-empty-without-user-message",
			messages: []assistant.Message{
				{Role: assistant.ChatRole_System, Content: "system"},
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

func TestParseSelectedSkillDirectives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single-directive-with-text",
			input: "/web-research find events in maple ridge",
			want:  []string{"web-research"},
		},
		{
			name:  "multiple-directives",
			input: "/todo-update /todo-summary check this",
			want:  []string{"todo-update", "todo-summary"},
		},
		{
			name:  "deduplicates-and-normalizes-case",
			input: "/Todo-Update /todo-update update this",
			want:  []string{"todo-update"},
		},
		{
			name:  "ignores-invalid-leading-directive",
			input: "/todo:update /todo-update update this",
			want:  []string{"todo-update"},
		},
		{
			name:  "returns-empty-when-no-leading-directive",
			input: "search /web-research now",
			want:  nil,
		},
		{
			name:  "handles-punctuation-suffix",
			input: "/web-research, find events",
			want:  []string{"web-research"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseSelectedSkillDirectives(tt.input))
		})
	}
}
