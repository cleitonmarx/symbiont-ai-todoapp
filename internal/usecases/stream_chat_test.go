package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain/mocks"
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
		setupMocks      func(*mocks.MockChatMessageRepository, *mocks.MockTodoRepository, *mocks.MockLLMClient)
		expectErr       bool
		expectedContent string
	}{
		"success-with-usage": {
			userMessage: "Hello, how are you?",
			setupMocks: func(chatRepo *mocks.MockChatMessageRepository, todoRepo *mocks.MockTodoRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					Embed(mock.Anything, "test-embedding-model", "Hello, how are you?").
					Return([]float64{0.1, 0.2, 0.3}, nil)

				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 30, mock.Anything).
					Run(func(ctx context.Context, page, pageSize int, opts ...domain.ListTodoOptions) {
						// Verify that the embedding option is provided
						params := &domain.ListTodosParams{}
						for _, opt := range opts {
							opt(params)
						}
						assert.NotNil(t, params.Embedding)
					}).
					Return([]domain.Todo{{Title: "Test Todo"}}, false, nil)

				// history: empty
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, mock.Anything).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						// Simulate events
						_ = onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "I'm "})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "doing "})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "great!"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
							Usage: &domain.LLMUsage{
								PromptTokens:     10,
								CompletionTokens: 5,
								TotalTokens:      15,
							},
						})
					}).
					Return(nil)

				// user and assistant saves...
				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_User
					})).
					Run(func(ctx context.Context, msg domain.ChatMessage) {
						assert.Equal(t, userMsgID, msg.ID)
						assert.Equal(t, "Hello, how are you?", msg.Content)
					}).
					Return(nil).
					Once()

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole("assistant")
					})).
					Run(func(ctx context.Context, msg domain.ChatMessage) {
						assert.Equal(t, assistantMsgID, msg.ID)
						assert.Equal(t, "I'm doing great!", msg.Content)
						assert.Equal(t, 10, msg.PromptTokens)
						assert.Equal(t, 5, msg.CompletionTokens)
					}).
					Return(nil).
					Once()
			},
			expectErr:       false,
			expectedContent: "I'm doing great!",
		},
		"success-without-usage": {
			userMessage: "Test",
			setupMocks: func(chatRepo *mocks.MockChatMessageRepository, todoRepo *mocks.MockTodoRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					Embed(mock.Anything, "test-embedding-model", "Test").
					Return([]float64{0.1, 0.2, 0.3}, nil)

				todoRepo.EXPECT().
					ListTodos(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]domain.Todo{}, false, nil)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, mock.Anything).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
							Usage:              nil,
						})
					}).
					Return(nil)

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_User
					})).
					Return(nil).
					Once()

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_Assistant
					})).
					Run(func(ctx context.Context, msg domain.ChatMessage) {
						assert.Equal(t, 0, msg.PromptTokens)
						assert.Equal(t, 0, msg.CompletionTokens)
					}).
					Return(nil).
					Once()
			},
			expectErr:       false,
			expectedContent: "OK",
		},
		"list-todos-error": {
			userMessage: "Test",
			setupMocks: func(chatRepo *mocks.MockChatMessageRepository, todoRepo *mocks.MockTodoRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					Embed(mock.Anything, "test-embedding-model", "Test").
					Return([]float64{0.1, 0.2, 0.3}, nil)

				todoRepo.EXPECT().
					ListTodos(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, false, errors.New("list todos error"))
			},
			expectErr: true,
		},
		"llm-embed-error": {
			userMessage: "Test",
			setupMocks: func(chatRepo *mocks.MockChatMessageRepository, todoRepo *mocks.MockTodoRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					Embed(mock.Anything, "test-embedding-model", "Test").
					Return(nil, errors.New("embed error"))
			},
			expectErr: true,
		},
		"llm-chat-error": {
			userMessage: "Test",
			setupMocks: func(chatRepo *mocks.MockChatMessageRepository, todoRepo *mocks.MockTodoRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					Embed(mock.Anything, "test-embedding-model", "Test").
					Return([]float64{0.1, 0.2, 0.3}, nil)

				todoRepo.EXPECT().
					ListTodos(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]domain.Todo{}, false, nil)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, mock.Anything).
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
			setupMocks: func(chatRepo *mocks.MockChatMessageRepository, todoRepo *mocks.MockTodoRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					Embed(mock.Anything, "test-embedding-model", "Test").
					Return([]float64{0.1, 0.2, 0.3}, nil)

				todoRepo.EXPECT().
					ListTodos(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]domain.Todo{}, false, nil)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, mock.Anything).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_User
					})).
					Return(errors.New("db error")).
					Once()
			},
			expectErr: true,
		},
		"assistant-message-save-error": {
			userMessage: "Test",
			setupMocks: func(chatRepo *mocks.MockChatMessageRepository, todoRepo *mocks.MockTodoRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					Embed(mock.Anything, "test-embedding-model", "Test").
					Return([]float64{0.1, 0.2, 0.3}, nil)

				todoRepo.EXPECT().
					ListTodos(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]domain.Todo{}, false, nil)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, mock.Anything).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_User
					})).
					Return(nil).
					Once()

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_Assistant
					})).
					Return(errors.New("db error")).
					Once()
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			chatRepo := mocks.NewMockChatMessageRepository(t)
			todoRepo := mocks.NewMockTodoRepository(t)
			client := mocks.NewMockLLMClient(t)

			tt.setupMocks(chatRepo, todoRepo, client)

			useCase := NewStreamChatImpl(chatRepo, todoRepo, client, "test-model", "test-embedding-model")

			var capturedContent string
			err := useCase.Execute(context.Background(), tt.userMessage, func(eventType domain.LLMStreamEventType, data any) error {
				if eventType == domain.LLMStreamEventType_Delta {
					delta := data.(domain.LLMStreamEventDelta)
					capturedContent += delta.Text
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
