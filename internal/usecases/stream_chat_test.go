package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStreamChatImpl_Execute(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userMsgID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	assistantMsgID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)
	promptTokens := 11
	completionTokens := 7
	totalTokens := 18

	tests := map[string]streamChatTestTableEntry{
		"success": {
			userMessage:              "Hello, how are you?",
			model:                    "test-model",
			customSummaryExpectation: true,
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{
						ID:                  uuid.MustParse("323e4567-e89b-12d3-a456-426614174002"),
						ConversationID:      conversationID,
						CurrentStateSummary: "Current intent: organize todos",
					}, true, nil).
					Once()

				actionRegistry.EXPECT().
					List().
					Return([]domain.AssistantActionDefinition{})

				expectNowCalls(timeProvider, fixedTime, 5)

				// history: empty
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{
						{
							ID:        uuid.New(),
							ChatRole:  domain.ChatRole_User,
							Content:   "Previous message",
							CreatedAt: fixedTime.Add(-time.Minute),
						},
					}, false, nil).
					Once()

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) {
						foundSummaryContext := false
						for _, msg := range req.Messages {
							if msg.Role == domain.ChatRole_Developer && strings.Contains(msg.Content, "Current intent: organize todos") {
								foundSummaryContext = true
								break
							}
						}
						assert.True(t, foundSummaryContext)

						// Simulate events
						_ = onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
						_ = onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "I'm "})
						_ = onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "doing "})
						_ = onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "great!"})
						_ = onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
							Usage: domain.AssistantUsage{
								PromptTokens:     promptTokens,
								CompletionTokens: completionTokens,
								TotalTokens:      totalTokens,
							},
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Hello, how are you?",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:             domain.ChatRole_Assistant,
						Content:          "I'm doing great!",
						ID:               &assistantMsgID,
						PromptTokens:     &promptTokens,
						CompletionTokens: &completionTokens,
						TotalTokens:      &totalTokens,
						ToolCallsLen:     0,
						HasToolCallID:    false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "I'm doing great!",
		},
		"success-with-new-conversation": {
			userMessage:              "Hello, how are you?",
			model:                    "test-model",
			customSummaryExpectation: true,
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {

				conversationRepo.EXPECT().
					CreateConversation(
						mock.Anything,
						"Hello, how are you?",
						domain.ConversationTitleSource_Auto,
					).
					Return(domain.Conversation{
						ID:          conversationID,
						Title:       "Hello, how are you?",
						TitleSource: domain.ConversationTitleSource_Auto,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					}, nil)

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{
						ID:                  uuid.MustParse("323e4567-e89b-12d3-a456-426614174002"),
						ConversationID:      conversationID,
						CurrentStateSummary: "Current intent: organize todos",
					}, true, nil).
					Once()

				actionRegistry.EXPECT().
					List().
					Return([]domain.AssistantActionDefinition{})

				expectNowCalls(timeProvider, fixedTime, 5)

				// history: empty
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{
						{
							ID:        uuid.New(),
							ChatRole:  domain.ChatRole_User,
							Content:   "Previous message",
							CreatedAt: fixedTime.Add(-time.Minute),
						},
					}, false, nil).
					Once()

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) {
						foundSummaryContext := false
						for _, msg := range req.Messages {
							if msg.Role == domain.ChatRole_Developer && strings.Contains(msg.Content, "Current intent: organize todos") {
								foundSummaryContext = true
								break
							}
						}
						assert.True(t, foundSummaryContext)

						// Simulate events
						_ = onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
						_ = onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "I'm "})
						_ = onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "doing "})
						_ = onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "great!"})
						_ = onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
							Usage: domain.AssistantUsage{
								PromptTokens:     promptTokens,
								CompletionTokens: completionTokens,
								TotalTokens:      totalTokens,
							},
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Hello, how are you?",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:             domain.ChatRole_Assistant,
						Content:          "I'm doing great!",
						ID:               &assistantMsgID,
						PromptTokens:     &promptTokens,
						CompletionTokens: &completionTokens,
						TotalTokens:      &totalTokens,
						ToolCallsLen:     0,
						HasToolCallID:    false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "I'm doing great!",
		},
		"assistant-empty-response": {
			userMessage: "Test",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				actionRegistry.EXPECT().
					List().
					Return([]domain.AssistantActionDefinition{})

				expectNowCalls(timeProvider, fixedTime, 5)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) {
						_ = onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
						_ = onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Test",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "Sorry, I could not process your request. Please try again.",
						ID:            &assistantMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "Sorry, I could not process your request. Please try again.\n",
		},
		"empty-user-message": {
			userMessage: "   ",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			expectErr: true,
		},
		"empty-model": {
			userMessage: "Hello",
			model:       "",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			expectErr: true,
		},
		"list-chat-history-error": {
			userMessage: "Test",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				expectNowCalls(timeProvider, fixedTime, 2)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return(nil, false, errors.New("history error")).
					Once()
			},
			expectErr: true,
		},
		"onEvent-meta-error": {
			userMessage: "Test",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				actionRegistry.EXPECT().
					List().
					Return([]domain.AssistantActionDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
						return onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Test",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "",
						ID:            &assistantMsgID,
						MessageState:  domain.ChatMessageState_Failed,
						ErrorMessage:  &onEventErr,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				})
			},
			expectErr:      true,
			onEventErrType: domain.AssistantEventType_TurnStarted,
		},
		"onEvent-delta-error": {
			userMessage: "Test",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				actionRegistry.EXPECT().
					List().
					Return([]domain.AssistantActionDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
						if err := onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						return onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "Hi"})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Test",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "",
						ID:            &assistantMsgID,
						MessageState:  domain.ChatMessageState_Failed,
						ErrorMessage:  &onEventErr,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				})
			},
			expectErr:      true,
			onEventErrType: domain.AssistantEventType_MessageDelta,
		},
		"llm-chatstream-error": {
			userMessage: "Test",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()
				actionRegistry.EXPECT().
					List().
					Return([]domain.AssistantActionDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("llm error"))

				llmErr := "llm error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Test",
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "",
						MessageState:  domain.ChatMessageState_Failed,
						ErrorMessage:  &llmErr,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				})
			},
			expectErr: true,
		},
		"chatstream-without-meta-persists-user-after-loop": {
			userMessage: "No meta",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				actionRegistry.EXPECT().
					List().
					Return([]domain.AssistantActionDefinition{})

				expectNowCalls(timeProvider, fixedTime, 5)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) {
						_ = onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "Hello from model"})
						_ = onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
							AssistantMessageID: "",
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "No meta",
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "Hello from model",
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "Hello from model",
		},
		"user-message-save-error": {
			userMessage: "Test",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				actionRegistry.EXPECT().
					List().
					Return([]domain.AssistantActionDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
						if err := onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						if err := onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "OK"}); err != nil {
							return err
						}
						return onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					})

				dbErr := errors.New("db error")
				dbErrText := dbErr.Error()
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Test",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
						CreateErr:     dbErr,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "",
						ID:            &assistantMsgID,
						MessageState:  domain.ChatMessageState_Failed,
						ErrorMessage:  &dbErrText,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				})
			},
			expectErr: true,
		},
		"assistant-message-save-error": {
			userMessage: "Test",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
				actionRegistry *domain.MockAssistantActionRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				actionRegistry.EXPECT().
					List().
					Return([]domain.AssistantActionDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
						if err := onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						if err := onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "OK"}); err != nil {
							return err
						}
						return onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					})

				dbErr := errors.New("db error")
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Test",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "OK",
						ID:            &assistantMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
						CreateErr:     dbErr,
					},
				})
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testStreamChatImpl(t, tt)
		})
	}
}

func TestInitStreamChat_Initialize(t *testing.T) {
	i := InitStreamChat{}

	ctx, err := i.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify that the StreamChat use case is registered
	streamChatUseCase, err := depend.Resolve[StreamChat]()
	assert.NoError(t, err)
	assert.NotNil(t, streamChatUseCase)
}
