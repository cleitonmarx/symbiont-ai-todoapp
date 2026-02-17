package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConversationSummary_CurrentStateOrDefault(t *testing.T) {
	tests := map[string]struct {
		summary ConversationSummary
		want    string
	}{
		"uses-default-when-empty": {
			summary: ConversationSummary{},
			want:    DefaultConversationStateSummary,
		},
		"uses-default-when-whitespace": {
			summary: ConversationSummary{CurrentStateSummary: "   "},
			want:    DefaultConversationStateSummary,
		},
		"returns-trimmed-summary": {
			summary: ConversationSummary{CurrentStateSummary: "  current state  "},
			want:    "current state",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.summary.CurrentStateOrDefault()
			assert.Equal(t, tt.want, got)
		})
	}
}
