package assistant

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestActionApprovalDecision_Validate(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	turnID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	now := time.Now()
	reason := "user denied"

	validKey := ActionApprovalKey{
		ConversationID: conversationID,
		TurnID:         turnID,
		ActionCallID:   "action-call-1",
	}

	tests := map[string]struct {
		decision ActionApprovalDecision
		wantErr  bool
		errMsg   string
	}{
		"valid-with-approved-status": {
			decision: ActionApprovalDecision{
				Key:        validKey,
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: false,
		},
		"valid-with-rejected-status": {
			decision: ActionApprovalDecision{
				Key:        validKey,
				ActionName: "DeleteTodo",
				Status:     ChatMessageApprovalStatus_Rejected,
				Reason:     &reason,
				DecidedAt:  now,
			},
			wantErr: false,
		},
		"valid-with-nil-reason": {
			decision: ActionApprovalDecision{
				Key:        validKey,
				ActionName: "CreateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				Reason:     nil,
				DecidedAt:  now,
			},
			wantErr: false,
		},
		"missing-conversation-id": {
			decision: ActionApprovalDecision{
				Key: ActionApprovalKey{
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
			decision: ActionApprovalDecision{
				Key: ActionApprovalKey{
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
			decision: ActionApprovalDecision{
				Key: ActionApprovalKey{
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
			decision: ActionApprovalDecision{
				Key: ActionApprovalKey{
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
			decision: ActionApprovalDecision{
				Key:        validKey,
				ActionName: "",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "action_name is required",
		},
		"action-name-only-whitespace": {
			decision: ActionApprovalDecision{
				Key:        validKey,
				ActionName: "   ",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "action_name is required",
		},
		"missing-decided-at": {
			decision: ActionApprovalDecision{
				Key:        validKey,
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus_Approved,
				DecidedAt:  time.Time{},
			},
			wantErr: true,
			errMsg:  "decided_at is required",
		},
		"invalid-status": {
			decision: ActionApprovalDecision{
				Key:        validKey,
				ActionName: "UpdateTodo",
				Status:     ChatMessageApprovalStatus("INVALID"),
				DecidedAt:  now,
			},
			wantErr: true,
			errMsg:  "invalid status: INVALID",
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
