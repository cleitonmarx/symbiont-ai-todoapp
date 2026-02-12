package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStreamChatImpl_Execute(t *testing.T) {
	userMsgID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	assistantMsgID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		userMessage     string
		model           string
		setExpectations func(
			*domain.MockChatMessageRepository,
			*domain.MockCurrentTimeProvider,
			*domain.MockLLMClient,
			*domain.MockLLMToolRegistry,
			*domain.MockUnitOfWork,
			*domain.MockOutboxRepository,
		)
		expectErr       bool
		expectedContent string
		onEventErrType  domain.LLMStreamEventType
	}{
		"success": {
			userMessage: "Hello, how are you?",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 5)

				// history: empty
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{
						{
							ID:        uuid.New(),
							ChatRole:  domain.ChatRole_User,
							Content:   "Previous message",
							CreatedAt: fixedTime.Add(-time.Minute),
						},
					}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						// Simulate events
						_ = onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "I'm "})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "doing "})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "great!"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Hello, how are you?",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "I'm doing great!",
						ID:            &assistantMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "I'm doing great!",
		},
		"assistant-empty-response": {
			userMessage: "Test",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 5)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, uow, outbox, fixedTime, []persistCallExpectation{
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
			expectErr:   true,
		},
		"empty-model": {
			userMessage: "Hello",
			model:       "",
			expectErr:   true,
		},
		"list-chat-history-error": {
			userMessage: "Test",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				expectNowCalls(timeProvider, fixedTime, 2)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return(nil, false, errors.New("history error")).
					Once()
			},
			expectErr: true,
		},
		"onEvent-meta-error": {
			userMessage: "Test",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
						return onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, uow, outbox, fixedTime, []persistCallExpectation{
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
			onEventErrType: domain.LLMStreamEventType_Meta,
		},
		"onEvent-delta-error": {
			userMessage: "Test",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
						if err := onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						return onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "Hi"})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, uow, outbox, fixedTime, []persistCallExpectation{
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
			onEventErrType: domain.LLMStreamEventType_Delta,
		},
		"llm-chatstream-error": {
			userMessage: "Test",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("llm error"))

				llmErr := "llm error"
				expectPersistSequence(t, chatRepo, uow, outbox, fixedTime, []persistCallExpectation{
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
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 5)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "Hello from model"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: "",
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, uow, outbox, fixedTime, []persistCallExpectation{
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
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
						if err := onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						if err := onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "OK"}); err != nil {
							return err
						}
						return onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					})

				dbErr := errors.New("db error")
				dbErrText := dbErr.Error()
				expectPersistSequence(t, chatRepo, uow, outbox, fixedTime, []persistCallExpectation{
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
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
						if err := onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						if err := onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "OK"}); err != nil {
							return err
						}
						return onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					})

				dbErr := errors.New("db error")
				expectPersistSequence(t, chatRepo, uow, outbox, fixedTime, []persistCallExpectation{
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

			chatRepo := domain.NewMockChatMessageRepository(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			llmClient := domain.NewMockLLMClient(t)
			lltToolRegistry := domain.NewMockLLMToolRegistry(t)
			uow := domain.NewMockUnitOfWork(t)
			outbox := domain.NewMockOutboxRepository(t)
			if tt.setExpectations != nil {
				tt.setExpectations(chatRepo, timeProvider, llmClient, lltToolRegistry, uow, outbox)
			}

			useCase := NewStreamChatImpl(chatRepo, timeProvider, llmClient, lltToolRegistry, uow, "test-embedding-model", 7)

			var capturedContent string
			err := useCase.Execute(context.Background(), tt.userMessage, tt.model, func(eventType domain.LLMStreamEventType, data any) error {
				if tt.onEventErrType != "" && eventType == tt.onEventErrType {
					return errors.New("onEvent error")
				}
				if eventType == domain.LLMStreamEventType_Delta {
					delta := data.(domain.LLMStreamEventDelta)
					capturedContent += delta.Text
				}
				if eventType == domain.LLMStreamEventType_ToolCall {
					fc := data.(domain.LLMStreamEventToolCall)
					capturedContent += fc.Text
				}
				return nil
			})

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedContent != "" {
					assert.Equal(t, tt.expectedContent, capturedContent)
				}
			}
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
