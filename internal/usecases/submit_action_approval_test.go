package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubmitActionApprovalImpl_Execute(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	turnID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	actionCallID := "call-1"

	tests := map[string]struct {
		input            SubmitActionApprovalInput
		publisherErr     error
		expectErr        bool
		expectedErrValue string
	}{
		"success-approved": {
			input: SubmitActionApprovalInput{
				ConversationID: conversationID,
				TurnID:         turnID,
				ActionCallID:   actionCallID,
				ActionName:     "delete_todo",
				Status:         domain.ChatMessageApprovalStatus_Approved,
				Reason:         common.Ptr("approved by user"),
			},
		},
		"validation-error-invalid-status": {
			input: SubmitActionApprovalInput{
				ConversationID: conversationID,
				TurnID:         turnID,
				ActionCallID:   actionCallID,
				ActionName:     "delete_todo",
				Status:         domain.ChatMessageApprovalStatus_Expired,
			},
			expectErr:        true,
			expectedErrValue: "status must be APPROVED or REJECTED",
		},
		"publish-error": {
			input: SubmitActionApprovalInput{
				ConversationID: conversationID,
				TurnID:         turnID,
				ActionCallID:   actionCallID,
				ActionName:     "delete_todo",
				Status:         domain.ChatMessageApprovalStatus_Rejected,
			},
			publisherErr: errors.New("pubsub unavailable"),
			expectErr:    true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			publisher := domain.NewMockEventPublisher(t)
			if !tt.expectErr || tt.publisherErr != nil {
				publisher.EXPECT().
					PublishEvent(mock.Anything, mock.MatchedBy(func(event domain.OutboxEvent) bool {
						if event.Topic != domain.OutboxTopic_ActionApprovals {
							return false
						}
						if event.EventType != domain.EventType_ACTION_APPROVAL_DECIDED {
							return false
						}
						if event.EntityType != domain.OutboxEntityType_ChatMessage {
							return false
						}
						var payload domain.AssistantActionApprovalDecision
						if err := json.Unmarshal(event.Payload, &payload); err != nil {
							return false
						}
						return payload.Key.ConversationID == tt.input.ConversationID &&
							payload.Key.TurnID == tt.input.TurnID &&
							payload.Key.ActionCallID == tt.input.ActionCallID &&
							payload.Status == tt.input.Status
					})).
					Return(tt.publisherErr).
					Once()
			}

			uc := NewSubmitActionApprovalImpl(publisher)
			err := uc.Execute(context.Background(), tt.input)

			if !tt.expectErr {
				assert.NoError(t, err)
				return
			}

			assert.Error(t, err)
			if tt.expectedErrValue != "" {
				assert.Equal(t, tt.expectedErrValue, err.Error())
			}
		})
	}
}

func TestInitSubmitActionApproval_Initialize(t *testing.T) {
	t.Parallel()

	publisher := domain.NewMockEventPublisher(t)
	init := InitSubmitActionApproval{
		Publisher: publisher,
	}

	_, err := init.Initialize(context.Background())
	assert.NoError(t, err)

	uc, err := depend.Resolve[SubmitActionApproval]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)
}
