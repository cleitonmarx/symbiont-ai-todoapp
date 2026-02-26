package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChatMessage_IsApprovalPending(t *testing.T) {
	t.Parallel()

	pending := ChatMessageApprovalStatus_Pending
	approved := ChatMessageApprovalStatus_Approved

	tests := map[string]struct {
		message ChatMessage
		want    bool
	}{
		"pending-status": {
			message: ChatMessage{
				ApprovalStatus: &pending,
			},
			want: true,
		},
		"approved-status": {
			message: ChatMessage{
				ApprovalStatus: &approved,
			},
			want: false,
		},
		"nil-status": {
			message: ChatMessage{},
			want:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.message.IsApprovalPending()
			assert.Equal(t, tt.want, got)
		})
	}
}
