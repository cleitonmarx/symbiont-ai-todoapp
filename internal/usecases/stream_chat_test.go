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

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

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

				uow.EXPECT().
					ChatMessage().Return(chatRepo)
				uow.EXPECT().
					Outbox().Return(outbox)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()

				outbox.EXPECT().
					CreateChatEvent(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, event domain.ChatMessageEvent) {
						assert.Equal(t, domain.EventType_CHAT_MESSAGE_SENT, event.Type)
						assert.Equal(t, false, event.IsToolSuccess)
					}).
					Return(nil).
					Twice()

				chatRepo.EXPECT().
					CreateChatMessages(mock.Anything, mock.MatchedBy(func(msgs []domain.ChatMessage) bool {
						return len(msgs) == 2
					})).
					Run(func(ctx context.Context, msgs []domain.ChatMessage) {
						assert.Equal(t, userMsgID, msgs[0].ID)
						assert.Equal(t, "Hello, how are you?", msgs[0].Content)
						assert.Equal(t, assistantMsgID, msgs[1].ID)
						assert.Equal(t, "I'm doing great!", msgs[1].Content)
					}).
					Return(nil).
					Once()

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

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

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

				uow.EXPECT().
					ChatMessage().Return(chatRepo)
				uow.EXPECT().
					Outbox().Return(outbox)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()

				outbox.EXPECT().
					CreateChatEvent(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, event domain.ChatMessageEvent) {
						assert.Equal(t, domain.EventType_CHAT_MESSAGE_SENT, event.Type)
					}).
					Return(nil).
					Times(2)

				chatRepo.EXPECT().
					CreateChatMessages(mock.Anything, mock.MatchedBy(func(msgs []domain.ChatMessage) bool {
						if len(msgs) != 2 {
							return false
						}
						return msgs[0].ChatRole == domain.ChatRole_User &&
							msgs[1].ChatRole == domain.ChatRole_Assistant &&
							msgs[1].Content == "Sorry, I could not process your request. Please try again."
					})).
					Return(nil).
					Once()
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
				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

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

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

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

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

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

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("llm error"))
			},
			expectErr: true,
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

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

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
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				uow.EXPECT().
					ChatMessage().Return(chatRepo)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()

				chatRepo.EXPECT().
					CreateChatMessages(mock.Anything, mock.MatchedBy(func(msgs []domain.ChatMessage) bool {
						return msgs[0].ChatRole == domain.ChatRole_User
					})).
					Return(errors.New("db error")).
					Once()
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

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

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
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				uow.EXPECT().
					ChatMessage().Return(chatRepo)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()

				chatRepo.EXPECT().
					CreateChatMessages(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, msgs []domain.ChatMessage) {
						assert.Equal(t, domain.ChatRole_User, msgs[0].ChatRole)
						assert.Equal(t, domain.ChatRole_Assistant, msgs[1].ChatRole)
					}).
					Return(errors.New("db error")).
					Once()
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
