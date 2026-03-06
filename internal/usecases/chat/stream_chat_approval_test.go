package chat

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
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

	actionDefinition := assistant.ActionDefinition{
		Name: actionName,
		Approval: assistant.ActionApproval{
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
		waitDecision               assistant.ActionApprovalDecision
		waitErr                    error
		expectedStatus             assistant.ChatMessageApprovalStatus
		expectedReason             *string
		expectedToolContent        string
		expectedToolMessageState   assistant.ChatMessageState
		expectedToolError          *string
		expectActionExecution      bool
		expectedActionCompletedErr *string
		expectedActionCompletedOK  bool
		expectedActionExecuted     *bool
		expectedEventSequence      []assistant.EventType
		expectActionStarted        bool
	}{
		"approved": {
			waitDecision: assistant.ActionApprovalDecision{
				Status:    assistant.ChatMessageApprovalStatus_Approved,
				Reason:    common.Ptr("approved by user"),
				DecidedAt: fixedTime,
			},
			expectedStatus:            assistant.ChatMessageApprovalStatus_Approved,
			expectedReason:            common.Ptr("approved by user"),
			expectedToolContent:       approvedToolContent,
			expectedToolMessageState:  assistant.ChatMessageState_Completed,
			expectActionExecution:     true,
			expectedActionCompletedOK: true,
			expectedActionExecuted:    common.Ptr(true),
			expectedEventSequence: []assistant.EventType{
				assistant.EventType_TurnStarted,
				assistant.EventType_ActionApprovalRequired,
				assistant.EventType_ActionApprovalResolved,
				assistant.EventType_ActionStarted,
				assistant.EventType_ActionCompleted,
				assistant.EventType_MessageDelta,
				assistant.EventType_TurnCompleted,
			},
			expectActionStarted: true,
		},
		"rejected": {
			waitDecision: assistant.ActionApprovalDecision{
				Status:    assistant.ChatMessageApprovalStatus_Rejected,
				Reason:    common.Ptr("user denied"),
				DecidedAt: fixedTime,
			},
			expectedStatus:             assistant.ChatMessageApprovalStatus_Rejected,
			expectedReason:             common.Ptr("user denied"),
			expectedToolContent:        approvalBlockedActionContent(assistant.ActionCall{ID: actionCallID, Name: actionName}, assistant.ChatMessageApprovalStatus_Rejected, "user denied"),
			expectedToolMessageState:   assistant.ChatMessageState_Failed,
			expectedToolError:          common.Ptr("user denied"),
			expectActionExecution:      false,
			expectedActionCompletedErr: common.Ptr("user denied"),
			expectedActionCompletedOK:  false,
			expectedActionExecuted:     common.Ptr(false),
			expectedEventSequence: []assistant.EventType{
				assistant.EventType_TurnStarted,
				assistant.EventType_ActionApprovalRequired,
				assistant.EventType_ActionApprovalResolved,
				assistant.EventType_ActionCompleted,
				assistant.EventType_MessageDelta,
				assistant.EventType_TurnCompleted,
			},
		},
		"expired": {
			waitErr:                    context.DeadlineExceeded,
			expectedStatus:             assistant.ChatMessageApprovalStatus_Expired,
			expectedReason:             common.Ptr("approval request expired"),
			expectedToolContent:        approvalBlockedActionContent(assistant.ActionCall{ID: actionCallID, Name: actionName}, assistant.ChatMessageApprovalStatus_Expired, "approval request expired"),
			expectedToolMessageState:   assistant.ChatMessageState_Failed,
			expectedToolError:          common.Ptr("approval request expired"),
			expectActionExecution:      false,
			expectedActionCompletedErr: common.Ptr("approval request expired"),
			expectedActionCompletedOK:  false,
			expectedActionExecuted:     common.Ptr(false),
			expectedEventSequence: []assistant.EventType{
				assistant.EventType_TurnStarted,
				assistant.EventType_ActionApprovalRequired,
				assistant.EventType_ActionApprovalResolved,
				assistant.EventType_ActionCompleted,
				assistant.EventType_MessageDelta,
				assistant.EventType_TurnCompleted,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			chatRepo := assistant.NewMockChatMessageRepository(t)
			summaryRepo := assistant.NewMockConversationSummaryRepository(t)
			conversationRepo := assistant.NewMockConversationRepository(t)
			timeProvider := core.NewMockCurrentTimeProvider(t)
			assist := assistant.NewMockAssistant(t)
			actionRegistry := assistant.NewMockActionRegistry(t)
			skillRegistry := assistant.NewMockSkillRegistry(t)
			approvalDispatcher := assistant.NewMockActionApprovalDispatcher(t)
			uow := transaction.NewMockUnitOfWork(t)
			outbox := outbox.NewMockRepository(t)

			actionRegistry.EXPECT().
				GetRenderer(mock.Anything).
				Return(nil, false).
				Maybe()

			skillRegistry.EXPECT().
				ListRelevant(mock.Anything, mock.Anything).
				Return([]assistant.SkillDefinition{}).
				Once()

			timeProvider.EXPECT().
				Now().
				Return(fixedTime)

			conversationRepo.EXPECT().
				GetConversation(mock.Anything, conversationID).
				Return(assistant.Conversation{ID: conversationID}, true, nil).
				Once()

			summaryRepo.EXPECT().
				GetConversationSummary(mock.Anything, conversationID).
				Return(assistant.ConversationSummary{}, false, nil).
				Once()

			chatRepo.EXPECT().
				ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
				Return([]assistant.ChatMessage{}, false, nil).
				Once()

			actionRegistry.EXPECT().
				GetDefinition(actionName).
				Return(actionDefinition, true).
				Once()

			actionRegistry.EXPECT().
				StatusMessage(actionName).
				Return("executing delete_todo...").
				Once()

			if tc.expectActionExecution {
				actionRegistry.EXPECT().
					Execute(
						mock.Anything,
						assistant.ActionCall{
							ID:    actionCallID,
							Name:  actionName,
							Input: actionInput,
							Text:  "executing delete_todo...",
						},
						mock.Anything,
					).
					Return(assistant.Message{
						Role:         assistant.ChatRole_Tool,
						ActionCallID: common.Ptr(actionCallID),
						Content:      approvedToolContent,
					}).
					Once()
			}

			waitKeyMatcher := mock.MatchedBy(func(key assistant.ActionApprovalKey) bool {
				return key.ConversationID == conversationID &&
					key.ActionCallID == actionCallID &&
					key.TurnID != uuid.Nil
			})
			if tc.waitErr != nil {
				approvalDispatcher.EXPECT().
					Wait(mock.Anything, waitKeyMatcher).
					Return(assistant.ActionApprovalDecision{}, tc.waitErr).
					Once()
			} else {
				approvalDispatcher.EXPECT().
					Wait(mock.Anything, waitKeyMatcher).
					Return(tc.waitDecision, nil).
					Once()
			}

			assistantCallCount := 0
			assist.EXPECT().
				RunTurn(mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
					if assistantCallCount == 0 {
						assistantCallCount++
						if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						return onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
							ID:    actionCallID,
							Name:  actionName,
							Input: actionInput,
						})
					}

					require.GreaterOrEqual(t, len(req.Messages), 2)
					last := req.Messages[len(req.Messages)-1]
					assert.Equal(t, assistant.ChatRole_Tool, last.Role)
					assert.Equal(t, tc.expectedToolContent, last.Content)

					if err := onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: finalAssistantResponse}); err != nil {
						return err
					}
					return onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
						AssistantMessageID: assistantMsgID.String(),
						CompletedAt:        fixedTime.Format(time.RFC3339),
					})
				}).
				Times(2)

			expectedApprovalDecidedAt := fixedTime
			expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
				{
					Role:            assistant.ChatRole_User,
					Content:         "Delete todo 1",
					ID:              &userMsgID,
					ActionCallsLen:  0,
					HasActionCallID: false,
				},
				{
					Role:            assistant.ChatRole_Assistant,
					Content:         "",
					ActionCallsLen:  1,
					HasActionCallID: false,
					FirstActionCallText: func() *string {
						msg := "executing delete_todo..."
						return &msg
					}(),
				},
				{
					Role:                   assistant.ChatRole_Tool,
					Content:                tc.expectedToolContent,
					MessageState:           tc.expectedToolMessageState,
					ErrorMessage:           tc.expectedToolError,
					ApprovalStatus:         &tc.expectedStatus,
					ApprovalDecisionReason: tc.expectedReason,
					ApprovalDecidedAt:      &expectedApprovalDecidedAt,
					ActionCallsLen:         0,
					HasActionCallID:        true,
					ActionExecuted:         tc.expectedActionExecuted,
				},
				{
					Role:            assistant.ChatRole_Assistant,
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
				assist,
				actionRegistry,
				skillRegistry,
				approvalDispatcher,
				uow,
				"test-embedding-model",
				7,
			)

			var (
				eventSequence        []assistant.EventType
				approvalRequiredData assistant.ActionApprovalRequired
				approvalResolvedData assistant.ActionApprovalResolved
				actionCompletedData  assistant.ActionCompleted
				actionStartedSeen    bool
			)
			err := useCase.Execute(context.Background(), "Delete todo 1", "test-model", func(_ context.Context, eventType assistant.EventType, data any) error {
				eventSequence = append(eventSequence, eventType)
				switch eventType {
				case assistant.EventType_ActionApprovalRequired:
					approvalRequiredData = data.(assistant.ActionApprovalRequired)
				case assistant.EventType_ActionApprovalResolved:
					approvalResolvedData = data.(assistant.ActionApprovalResolved)
				case assistant.EventType_ActionCompleted:
					actionCompletedData = data.(assistant.ActionCompleted)
				case assistant.EventType_ActionStarted:
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
			assert.Equal(t, tc.expectedActionExecuted, actionCompletedData.ActionExecuted)
			assert.Equal(t, &tc.expectedStatus, actionCompletedData.ApprovalStatus)
			require.NotNil(t, actionCompletedData.OutputPreview)
			assert.Equal(t, tc.expectedToolContent, *actionCompletedData.OutputPreview)
			assert.False(t, actionCompletedData.OutputTruncated)

			if tc.expectActionStarted {
				assert.True(t, actionStartedSeen)
			} else {
				assert.False(t, actionStartedSeen)
				actionRegistry.AssertNotCalled(t, "Execute", mock.Anything, mock.Anything, mock.Anything)
			}

			if tc.waitErr != nil {
				assert.True(t, errors.Is(tc.waitErr, context.DeadlineExceeded))
			}
		})
	}
}
