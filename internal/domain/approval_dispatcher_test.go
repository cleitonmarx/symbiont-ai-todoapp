package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAssistantActionApprovalDecision_Validate(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	turnID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	now := time.Now()
	reason := "user denied"

	validKey := AssistantActionApprovalKey{
		ConversationID: conversationID,
		TurnID:         turnID,
		ActionCallID:   "action-call-1",
	}

	tests := map[string]struct {
		decision AssistantActionApprovalDecision
		wantErr  bool
		errMsg   string
	}{
		"valid-with-approved-status": {
			decision: AssistantActionApprovalDecision{
				Key:        validKey,
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: false,
		},
		"valid-with-rejected-status": {
			decision: AssistantActionApprovalDecision{
				Key:        validKey,
				ActionName: "DeleteTodo",
				Status:     ChatMessageApprovalStatus_Rejected,
				Reason:     &reason,
				DecidedAt:  now,
			},
			wantErr: false,
		},
		"valid-with-nil-reason": {
			decision: AssistantActionApprovalDecision{
				Key:        validKey,
				ActionName: "CreateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				Reason:     nil,
				DecidedAt:  now,
			},
			wantErr: false,
		},
		"missing-conversation-id": {
			decision: AssistantActionApprovalDecision{
				Key: AssistantActionApprovalKey{
					ConversationID: uuid.Nil,
					TurnID:         turnID,
					ActionCallID:   "action-call-1",
				},
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "conversation_id is required",
		},
		"missing-turn-id": {
			decision: AssistantActionApprovalDecision{
				Key: AssistantActionApprovalKey{
					ConversationID: conversationID,
					TurnID:         uuid.Nil,
					ActionCallID:   "action-call-1",
				},
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "turn_id is required",
		},
		"missing-action-call-id": {
			decision: AssistantActionApprovalDecision{
				Key: AssistantActionApprovalKey{
					ConversationID: conversationID,
					TurnID:         turnID,
					ActionCallID:   "",
				},
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "action_call_id is required",
		},
		"action-call-id-only-whitespace": {
			decision: AssistantActionApprovalDecision{
				Key: AssistantActionApprovalKey{
					ConversationID: conversationID,
					TurnID:         turnID,
					ActionCallID:   "   ",
				},
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "action_call_id is required",
		},
		"missing-action-name": {
			decision: AssistantActionApprovalDecision{
				Key:        validKey,
				ActionName: "",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "action_name is required",
		},
		"action-name-only-whitespace": {
			decision: AssistantActionApprovalDecision{
				Key:        validKey,
				ActionName: "   ",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "action_name is required",
		},
		"missing-decided-at": {
			decision: AssistantActionApprovalDecision{
				Key:        validKey,
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  time.Time{},
			},
			wantErr: true,
			errMsg:  "decided_at is required",
		},
		"invalid-status": {
			decision: AssistantActionApprovalDecision{
				Key:        validKey,
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus("INVALID"),
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "status must be APPROVED or REJECTED",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.decision.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
