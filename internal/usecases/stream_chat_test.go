package usecases

import (
	"context"
	"errors"
	"fmt"
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

				// user and assistant saves...
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
		"success-with-function-call": {
			userMessage: "Call a tool",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
			) {
				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				toolRegistry.EXPECT().
					StatusMessage("list_todos").
					Return("calling list_todos")

				toolRegistry.EXPECT().
					Call(
						mock.Anything,
						domain.LLMStreamEventToolCall{
							ID:        "func-123",
							Function:  "list_todos",
							Arguments: "{\"page\": 1, \"page_size\": 5, \"search_term\": \"searchTerm\"}",
							Text:      "calling list_todos",
						},
						mock.MatchedBy(func(msgs []domain.LLMChatMessage) bool {
							return len(msgs) > 0 && msgs[len(msgs)-1].Content == "Call a tool"
						}),
					).
					Return(domain.LLMChatMessage{Role: domain.ChatRole_Tool})

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(
						toolFunctionCallback(userMsgID, assistantMsgID, fixedTime),
					)

				// user and assistant saves...
				chatRepo.EXPECT().
					CreateChatMessages(mock.Anything, mock.MatchedBy(func(msgs []domain.ChatMessage) bool {
						return assert.Equal(t, 4, len(msgs))
					})).
					Run(func(ctx context.Context, msgs []domain.ChatMessage) {
						assert.Equal(t, domain.ChatRole_User, msgs[0].ChatRole)
						assert.Equal(t, "Call a tool", msgs[0].Content)

						assert.Equal(t, domain.ChatRole_Assistant, msgs[1].ChatRole)
						assert.Len(t, msgs[1].ToolCalls, 1)

						assert.Equal(t, domain.ChatRole_Tool, msgs[2].ChatRole)
						assert.Equal(t, "func-123", *msgs[2].ToolCallID)

						assert.Equal(t, domain.ChatRole_Assistant, msgs[3].ChatRole)
						assert.Equal(t, "Tool called successfully.", msgs[3].Content)

					}).
					Return(nil).
					Once()
			},
			expectErr:       false,
			expectedContent: "",
		},
		"assistant-empty-response": {
			userMessage: "Test",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
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
		"onEvent-function-call-error": {
			userMessage: "Call tool",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
			) {
				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				toolRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n")

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
						return onEvent(domain.LLMStreamEventType_ToolCall, domain.LLMStreamEventToolCall{
							ID:        "func-1",
							Function:  "fetch_todos",
							Arguments: `{"page": 1}`,
						})
					})
			},
			expectErr:      true,
			onEventErrType: domain.LLMStreamEventType_ToolCall,
		},
		"llm-chatstream-error": {
			userMessage: "Test",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
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
		"max-tool-cycles-exceeded": {
			userMessage: "Keep calling tools",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
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

				// Expect 7 tool calls (max cycles is 7, on the 8th it stops)
				toolRegistry.EXPECT().
					StatusMessage(mock.Anything).
					Return("calling tool...\n").
					Times(7)

				toolRegistry.EXPECT().
					Call(mock.Anything, mock.Anything, mock.Anything).
					Return(domain.LLMChatMessage{Role: domain.ChatRole_Tool, Content: "tool result"}).
					Times(7)

				callCount := 0
				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
						if callCount == 0 {
							if err := onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
								UserMessageID:      userMsgID,
								AssistantMessageID: assistantMsgID,
							}); err != nil {
								return err
							}
						}

						callCount++

						return onEvent(domain.LLMStreamEventType_ToolCall, domain.LLMStreamEventToolCall{
							ID:        fmt.Sprintf("func-%d", callCount),
							Function:  "fetch_todos",
							Arguments: fmt.Sprintf(`{"page": %d}`, callCount),
						})
					}).
					Times(8)

				// Final assistant message - contains the warning only
				chatRepo.EXPECT().
					CreateChatMessages(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, msgs []domain.ChatMessage) {
						assert.Len(t, msgs, 16)
						assert.Equal(t, domain.ChatRole_User, msgs[0].ChatRole)
					}).
					Return(nil).
					Once()

			},
			expectErr:       false,
			expectedContent: "calling tool...\ncalling tool...\ncalling tool...\ncalling tool...\ncalling tool...\ncalling tool...\ncalling tool...\nSorry, I could not process your request. Please try again.\n",
		},
		"repeated-tool-call-loop": {
			userMessage: "Call the same tool repeatedly",
			model:       "test-model",
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
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

				toolRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n").
					Times(5)

				toolRegistry.EXPECT().
					Call(mock.Anything, mock.Anything, mock.Anything).
					Return(domain.LLMChatMessage{Role: domain.ChatRole_Tool, Content: "same result"}).
					Times(5)

				callCount := 0
				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
						if callCount == 0 {
							if err := onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
								UserMessageID:      userMsgID,
								AssistantMessageID: assistantMsgID,
							}); err != nil {
								return err
							}
						}

						callCount++
						return onEvent(domain.LLMStreamEventType_ToolCall, domain.LLMStreamEventToolCall{
							ID:        fmt.Sprintf("func-%d", callCount),
							Function:  "fetch_todos",
							Arguments: `{"page": 1}`,
						})
					}).
					Times(6)

				chatRepo.EXPECT().
					CreateChatMessages(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, msgs []domain.ChatMessage) {
						assert.Len(t, msgs, 12)
						assert.Equal(t, domain.ChatRole_User, msgs[0].ChatRole)
					}).
					Return(nil).
					Once()
			},
			expectErr:       false,
			expectedContent: "calling fetch_todos...\ncalling fetch_todos...\ncalling fetch_todos...\ncalling fetch_todos...\ncalling fetch_todos...\nSorry, I could not process your request. Please try again.\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			chatRepo := domain.NewMockChatMessageRepository(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			llmClient := domain.NewMockLLMClient(t)
			lltToolRegistry := domain.NewMockLLMToolRegistry(t)

			if tt.setExpectations != nil {
				tt.setExpectations(chatRepo, timeProvider, llmClient, lltToolRegistry)
			}

			useCase := NewStreamChatImpl(chatRepo, timeProvider, llmClient, lltToolRegistry, "test-embedding-model", 7)

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

func toolFunctionCallback(userMsgID, assistantMsgID uuid.UUID, fixedTime time.Time) func(_ context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
	return func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
		if err := onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
			UserMessageID:      userMsgID,
			AssistantMessageID: assistantMsgID,
		}); err != nil {
			return err
		}

		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Content == "Call a tool" {
			err := onEvent(domain.LLMStreamEventType_ToolCall, domain.LLMStreamEventToolCall{
				ID:        "func-123",
				Function:  "list_todos",
				Arguments: `{"page": 1, "page_size": 5, "search_term": "searchTerm"}`,
			})
			return err
		}

		if lastMsg.Role == domain.ChatRole_Tool {
			if err := onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "Tool called successfully."}); err != nil {
				return err
			}
		}

		if err := onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
			AssistantMessageID: assistantMsgID.String(),
			CompletedAt:        fixedTime.Format(time.RFC3339),
		}); err != nil {
			return err
		}
		return nil
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
