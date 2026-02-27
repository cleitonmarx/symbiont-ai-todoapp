package usecases

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStreamChatImpl_Execute_ActionApprovalFlows(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userMsgID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	assistantMsgID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedTime := time.Date(2026, 2, 23, 10, 0, 0, 0, time.UTC)

	actionCallID := "func-approval-1"
	actionName := "delete_todo"
	actionInput := `{"id":"todo-1"}`
	finalAssistantResponse := "Understood."
	approvedToolContent := "todo deleted"

	actionDefinition := domain.AssistantActionDefinition{
		Name: actionName,
		Approval: domain.AssistantActionApproval{
			Required:    true,
			Title:       "Approve destructive action",
			Description: "This action deletes one todo item.",
			PreviewFields: []string{
				"todos[].title",
			},
			Timeout: 30 * time.Second,
		},
	}

	tests := map[string]struct {
		waitDecision               domain.AssistantActionApprovalDecision
		waitErr                    error
		expectedStatus             domain.ChatMessageApprovalStatus
		expectedReason             *string
		expectedToolContent        string
		expectedToolMessageState   domain.ChatMessageState
		expectedToolError          *string
		expectActionExecution      bool
		expectedActionCompletedErr *string
		expectedActionCompletedOK  bool
		expectedEventSequence      []domain.AssistantEventType
		expectActionStarted        bool
	}{
		"approved": {
			waitDecision: domain.AssistantActionApprovalDecision{
				Status:    domain.ChatMessageApprovalStatus_Approved,
				Reason:    common.Ptr("approved by user"),
				DecidedAt: fixedTime,
			},
			expectedStatus:            domain.ChatMessageApprovalStatus_Approved,
			expectedReason:            common.Ptr("approved by user"),
			expectedToolContent:       approvedToolContent,
			expectedToolMessageState:  domain.ChatMessageState_Completed,
			expectActionExecution:     true,
			expectedActionCompletedOK: true,
			expectedEventSequence: []domain.AssistantEventType{
				domain.AssistantEventType_TurnStarted,
				domain.AssistantEventType_ActionApprovalRequired,
				domain.AssistantEventType_ActionApprovalResolved,
				domain.AssistantEventType_ActionStarted,
				domain.AssistantEventType_ActionCompleted,
				domain.AssistantEventType_MessageDelta,
				domain.AssistantEventType_TurnCompleted,
			},
			expectActionStarted: true,
		},
		"rejected": {
			waitDecision: domain.AssistantActionApprovalDecision{
				Status:    domain.ChatMessageApprovalStatus_Rejected,
				Reason:    common.Ptr("user denied"),
				DecidedAt: fixedTime,
			},
			expectedStatus:             domain.ChatMessageApprovalStatus_Rejected,
			expectedReason:             common.Ptr("user denied"),
			expectedToolContent:        approvalBlockedActionContent(domain.AssistantActionCall{ID: actionCallID, Name: actionName}, domain.ChatMessageApprovalStatus_Rejected, "user denied"),
			expectedToolMessageState:   domain.ChatMessageState_Failed,
			expectedToolError:          common.Ptr("user denied"),
			expectActionExecution:      false,
			expectedActionCompletedErr: common.Ptr("user denied"),
			expectedActionCompletedOK:  false,
			expectedEventSequence: []domain.AssistantEventType{
				domain.AssistantEventType_TurnStarted,
				domain.AssistantEventType_ActionApprovalRequired,
				domain.AssistantEventType_ActionApprovalResolved,
				domain.AssistantEventType_ActionCompleted,
				domain.AssistantEventType_MessageDelta,
				domain.AssistantEventType_TurnCompleted,
			},
		},
		"expired": {
			waitErr:                    context.DeadlineExceeded,
			expectedStatus:             domain.ChatMessageApprovalStatus_Expired,
			expectedReason:             common.Ptr("approval request expired"),
			expectedToolContent:        approvalBlockedActionContent(domain.AssistantActionCall{ID: actionCallID, Name: actionName}, domain.ChatMessageApprovalStatus_Expired, "approval request expired"),
			expectedToolMessageState:   domain.ChatMessageState_Failed,
			expectedToolError:          common.Ptr("approval request expired"),
			expectActionExecution:      false,
			expectedActionCompletedErr: common.Ptr("approval request expired"),
			expectedActionCompletedOK:  false,
			expectedEventSequence: []domain.AssistantEventType{
				domain.AssistantEventType_TurnStarted,
				domain.AssistantEventType_ActionApprovalRequired,
				domain.AssistantEventType_ActionApprovalResolved,
				domain.AssistantEventType_ActionCompleted,
				domain.AssistantEventType_MessageDelta,
				domain.AssistantEventType_TurnCompleted,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			chatRepo := domain.NewMockChatMessageRepository(t)
			summaryRepo := domain.NewMockConversationSummaryRepository(t)
			conversationRepo := domain.NewMockConversationRepository(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			assistant := domain.NewMockAssistant(t)
			actionRegistry := domain.NewMockAssistantActionRegistry(t)
			skillRegistry := domain.NewMockAssistantSkillRegistry(t)
			approvalDispatcher := domain.NewMockAssistantActionApprovalDispatcher(t)
			uow := domain.NewMockUnitOfWork(t)
			outbox := domain.NewMockOutboxRepository(t)

			skillRegistry.EXPECT().
				ListRelevant(mock.Anything, mock.Anything).
				Return([]domain.AssistantSkillDefinition{}).
				Once()

			timeProvider.EXPECT().
				Now().
				Return(fixedTime)

			conversationRepo.EXPECT().
				GetConversation(mock.Anything, conversationID).
				Return(domain.Conversation{ID: conversationID}, true, nil).
				Once()

			summaryRepo.EXPECT().
				GetConversationSummary(mock.Anything, conversationID).
				Return(domain.ConversationSummary{}, false, nil).
				Once()

			chatRepo.EXPECT().
				ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
				Return([]domain.ChatMessage{}, false, nil).
				Once()

			actionRegistry.EXPECT().
				GetDefinition(actionName).
				Return(actionDefinition, true).
				Once()

			if tc.expectActionExecution {
				actionRegistry.EXPECT().
					StatusMessage(actionName).
					Return("executing delete_todo...").
					Once()

				actionRegistry.EXPECT().
					Execute(
						mock.Anything,
						domain.AssistantActionCall{
							ID:    actionCallID,
							Name:  actionName,
							Input: actionInput,
							Text:  "executing delete_todo...",
						},
						mock.Anything,
					).
					Return(domain.AssistantMessage{
						Role:         domain.ChatRole_Tool,
						ActionCallID: common.Ptr(actionCallID),
						Content:      approvedToolContent,
					}).
					Once()
			}

			waitKeyMatcher := mock.MatchedBy(func(key domain.AssistantActionApprovalKey) bool {
				return key.ConversationID == conversationID &&
					key.ActionCallID == actionCallID &&
					key.TurnID != uuid.Nil
			})
			if tc.waitErr != nil {
				approvalDispatcher.EXPECT().
					Wait(mock.Anything, waitKeyMatcher).
					Return(domain.AssistantActionApprovalDecision{}, tc.waitErr).
					Once()
			} else {
				approvalDispatcher.EXPECT().
					Wait(mock.Anything, waitKeyMatcher).
					Return(tc.waitDecision, nil).
					Once()
			}

			assistantCallCount := 0
			assistant.EXPECT().
				RunTurn(mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
					if assistantCallCount == 0 {
						assistantCallCount++
						if err := onEvent(ctx, domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						return onEvent(ctx, domain.AssistantEventType_ActionRequested, domain.AssistantActionCall{
							ID:    actionCallID,
							Name:  actionName,
							Input: actionInput,
						})
					}

					require.GreaterOrEqual(t, len(req.Messages), 2)
					last := req.Messages[len(req.Messages)-1]
					assert.Equal(t, domain.ChatRole_Tool, last.Role)
					assert.Equal(t, tc.expectedToolContent, last.Content)

					if err := onEvent(ctx, domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: finalAssistantResponse}); err != nil {
						return err
					}
					return onEvent(ctx, domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
						AssistantMessageID: assistantMsgID.String(),
						CompletedAt:        fixedTime.Format(time.RFC3339),
					})
				}).
				Times(2)

			expectedApprovalDecidedAt := fixedTime
			expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
				{
					Role:            domain.ChatRole_User,
					Content:         "Delete todo 1",
					ID:              &userMsgID,
					ActionCallsLen:  0,
					HasActionCallID: false,
				},
				{
					Role:            domain.ChatRole_Assistant,
					Content:         "",
					ActionCallsLen:  1,
					HasActionCallID: false,
				},
				{
					Role:                   domain.ChatRole_Tool,
					Content:                tc.expectedToolContent,
					MessageState:           tc.expectedToolMessageState,
					ErrorMessage:           tc.expectedToolError,
					ApprovalStatus:         &tc.expectedStatus,
					ApprovalDecisionReason: tc.expectedReason,
					ApprovalDecidedAt:      &expectedApprovalDecidedAt,
					ActionCallsLen:         0,
					HasActionCallID:        true,
				},
				{
					Role:            domain.ChatRole_Assistant,
					Content:         finalAssistantResponse,
					ID:              &assistantMsgID,
					ActionCallsLen:  0,
					HasActionCallID: false,
				},
			})

			useCase := NewStreamChatImpl(
				log.New(io.Discard, "", 0),
				chatRepo,
				summaryRepo,
				conversationRepo,
				timeProvider,
				assistant,
				actionRegistry,
				skillRegistry,
				approvalDispatcher,
				uow,
				"test-embedding-model",
				7,
			)

			var (
				eventSequence        []domain.AssistantEventType
				approvalRequiredData domain.AssistantActionApprovalRequired
				approvalResolvedData domain.AssistantActionApprovalResolved
				actionCompletedData  domain.AssistantActionCompleted
				actionStartedSeen    bool
			)
			err := useCase.Execute(context.Background(), "Delete todo 1", "test-model", func(_ context.Context, eventType domain.AssistantEventType, data any) error {
				eventSequence = append(eventSequence, eventType)
				switch eventType {
				case domain.AssistantEventType_ActionApprovalRequired:
					approvalRequiredData = data.(domain.AssistantActionApprovalRequired)
				case domain.AssistantEventType_ActionApprovalResolved:
					approvalResolvedData = data.(domain.AssistantActionApprovalResolved)
				case domain.AssistantEventType_ActionCompleted:
					actionCompletedData = data.(domain.AssistantActionCompleted)
				case domain.AssistantEventType_ActionStarted:
					actionStartedSeen = true
				}
				return nil
			}, WithConversationID(conversationID))
			require.NoError(t, err)

			assert.Equal(t, tc.expectedEventSequence, eventSequence)
			assert.Equal(t, conversationID, approvalRequiredData.ConversationID)
			assert.NotEqual(t, uuid.Nil, approvalRequiredData.TurnID)
			assert.Equal(t, actionCallID, approvalRequiredData.ActionCallID)
			assert.Equal(t, actionName, approvalRequiredData.Name)
			assert.Equal(t, actionInput, approvalRequiredData.Input)
			assert.Equal(t, actionDefinition.Approval.Title, approvalRequiredData.Title)
			assert.Equal(t, actionDefinition.Approval.Description, approvalRequiredData.Description)
			assert.Equal(t, actionDefinition.Approval.PreviewFields, approvalRequiredData.PreviewFields)
			assert.Equal(t, actionDefinition.Approval.Timeout, approvalRequiredData.Timeout)

			assert.Equal(t, conversationID, approvalResolvedData.ConversationID)
			assert.Equal(t, approvalRequiredData.TurnID, approvalResolvedData.TurnID)
			assert.Equal(t, actionCallID, approvalResolvedData.ActionCallID)
			assert.Equal(t, actionName, approvalResolvedData.Name)
			assert.Equal(t, tc.expectedStatus, approvalResolvedData.Status)
			assert.Equal(t, tc.expectedReason, approvalResolvedData.Reason)

			assert.Equal(t, actionCallID, actionCompletedData.ID)
			assert.Equal(t, actionName, actionCompletedData.Name)
			assert.Equal(t, tc.expectedActionCompletedOK, actionCompletedData.Success)
			assert.Equal(t, tc.expectedActionCompletedErr, actionCompletedData.Error)

			if tc.expectActionStarted {
				assert.True(t, actionStartedSeen)
			} else {
				assert.False(t, actionStartedSeen)
				actionRegistry.AssertNotCalled(t, "StatusMessage", mock.Anything)
				actionRegistry.AssertNotCalled(t, "Execute", mock.Anything, mock.Anything, mock.Anything)
			}

			if tc.waitErr != nil {
				assert.True(t, errors.Is(tc.waitErr, context.DeadlineExceeded))
			}
		})
	}
}
