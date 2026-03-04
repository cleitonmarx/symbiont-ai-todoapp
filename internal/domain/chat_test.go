package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChatMessage_IsActionCallSuccess(t *testing.T) {
	t.Parallel()

	actionCallID := "call-1"
	executed := true
	notExecuted := false

	tests := map[string]struct {
		message ChatMessage
		want    bool
	}{
		"successful-tool-result": {
			message: ChatMessage{
				ChatRole:       ChatRole_Tool,
				ActionCallID:   &actionCallID,
				ActionExecuted: &executed,
				MessageState:   ChatMessageState_Completed,
			},
			want: true,
		},
		"not-a-tool-message": {
			message: ChatMessage{
				ChatRole:       ChatRole_Assistant,
				ActionCallID:   &actionCallID,
				ActionExecuted: &executed,
				MessageState:   ChatMessageState_Completed,
			},
			want: false,
		},
		"missing-action-call-id": {
			message: ChatMessage{
				ChatRole:       ChatRole_Tool,
				ActionExecuted: &executed,
				MessageState:   ChatMessageState_Completed,
			},
			want: false,
		},
		"blocked-before-execution": {
			message: ChatMessage{
				ChatRole:       ChatRole_Tool,
				ActionCallID:   &actionCallID,
				ActionExecuted: &notExecuted,
				MessageState:   ChatMessageState_Completed,
			},
			want: false,
		},
		"failed-result": {
			message: ChatMessage{
				ChatRole:       ChatRole_Tool,
				ActionCallID:   &actionCallID,
				ActionExecuted: &executed,
				MessageState:   ChatMessageState_Failed,
			},
			want: false,
		},
		"missing-execution-state": {
			message: ChatMessage{
				ChatRole:     ChatRole_Tool,
				ActionCallID: &actionCallID,
				MessageState: ChatMessageState_Completed,
			},
			want: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.message.IsActionCallSuccess()
			assert.Equal(t, tt.want, got)
		})
	}
}

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
