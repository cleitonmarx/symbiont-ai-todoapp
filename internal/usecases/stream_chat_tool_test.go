package usecases

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestStreamChatImpl_Execute_ToolCases(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userMsgID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	assistantMsgID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]streamChatTestTableEntry{
		"success-with-function-call": {
			userMessage: "Call a tool",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

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
					Return(domain.LLMChatMessage{Role: domain.ChatRole_Tool, ToolCallID: common.Ptr("func-123")})

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				expectNowCalls(timeProvider, fixedTime, 7)

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(
						toolFunctionCallback(userMsgID, assistantMsgID, fixedTime),
					)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Call a tool",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "",
						ToolCallsLen:  1,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Tool,
						Content:       "",
						ToolCallsLen:  0,
						HasToolCallID: true,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "Tool called successfully.",
						ID:            &assistantMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "",
		},
		"tool-message-marked-as-failed-when-content-has-error": {
			userMessage: "Call failing tool",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				toolRegistry.EXPECT().
					StatusMessage("failing_tool").
					Return("calling failing_tool...\n")

				toolRegistry.EXPECT().
					Call(
						mock.Anything,
						domain.LLMStreamEventToolCall{
							ID:        "func-error",
							Function:  "failing_tool",
							Arguments: "{\"input\":\"x\"}",
							Text:      "calling failing_tool...\n",
						},
						mock.MatchedBy(func(msgs []domain.LLMChatMessage) bool {
							return len(msgs) > 0 && msgs[len(msgs)-1].Content == "Call failing tool"
						}),
					).
					Return(domain.LLMChatMessage{
						Role:       domain.ChatRole_Tool,
						ToolCallID: common.Ptr("func-error"),
						Content:    "error: failing_tool unavailable",
					})

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				expectNowCalls(timeProvider, fixedTime, 7)

				callCount := 0
				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
						if callCount == 0 {
							callCount++
							if err := onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
								UserMessageID:      userMsgID,
								AssistantMessageID: assistantMsgID,
							}); err != nil {
								return err
							}
							return onEvent(domain.LLMStreamEventType_ToolCall, domain.LLMStreamEventToolCall{
								ID:        "func-error",
								Function:  "failing_tool",
								Arguments: "{\"input\":\"x\"}",
							})
						}

						if err := onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "I could not complete that tool call."}); err != nil {
							return err
						}
						return onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Times(2)

				toolErr := "error: failing_tool unavailable"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Call failing tool",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "",
						ToolCallsLen:  1,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Tool,
						MessageState:  domain.ChatMessageState_Failed,
						ErrorMessage:  &toolErr,
						Content:       toolErr,
						ToolCallsLen:  0,
						HasToolCallID: true,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "I could not complete that tool call.",
						ID:            &assistantMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "calling failing_tool...\nI could not complete that tool call.",
		},
		"onEvent-function-call-error": {
			userMessage: "Call tool",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				toolRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n")

				expectNowCalls(timeProvider, fixedTime, 5)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
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

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Call tool",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
					{
						Role:          domain.ChatRole_Assistant,
						Content:       "",
						ToolCallsLen:  1,
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
			onEventErrType: domain.LLMStreamEventType_ToolCall,
		},
		"max-tool-cycles-exceeded": {
			userMessage: "Keep calling tools",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 19)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				toolRegistry.EXPECT().
					StatusMessage(mock.Anything).
					Return("calling tool...\n").
					Times(7)

				toolRegistry.EXPECT().
					Call(mock.Anything, mock.Anything, mock.Anything).
					Return(domain.LLMChatMessage{Role: domain.ChatRole_Tool, Content: "tool result", ToolCallID: common.Ptr("func-123")}).
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

				expectations := []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Keep calling tools",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				}
				for i := 0; i < 7; i++ {
					expectations = append(expectations,
						persistCallExpectation{
							Role:          domain.ChatRole_Assistant,
							Content:       "",
							ToolCallsLen:  1,
							HasToolCallID: false,
						},
						persistCallExpectation{
							Role:          domain.ChatRole_Tool,
							Content:       "tool result",
							ToolCallsLen:  0,
							HasToolCallID: true,
						},
					)
				}
				expectations = append(expectations, persistCallExpectation{
					Role:          domain.ChatRole_Assistant,
					Content:       "Sorry, I could not process your request. Please try again.",
					ID:            &assistantMsgID,
					ToolCallsLen:  0,
					HasToolCallID: false,
				})
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, expectations)
			},
			expectErr:       false,
			expectedContent: "calling tool...\ncalling tool...\ncalling tool...\ncalling tool...\ncalling tool...\ncalling tool...\ncalling tool...\nSorry, I could not process your request. Please try again.\n",
		},
		"repeated-tool-call-loop": {
			userMessage: "Call the same tool repeatedly",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				conversationRepo *domain.MockConversationRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *domain.MockLLMToolRegistry,
				uow *domain.MockUnitOfWork,
				outbox *domain.MockOutboxRepository,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMToolDefinition{})

				expectNowCalls(timeProvider, fixedTime, 15)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				toolRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n").
					Times(5)

				toolRegistry.EXPECT().
					Call(mock.Anything, mock.Anything, mock.Anything).
					Return(domain.LLMChatMessage{Role: domain.ChatRole_Tool, Content: "same result", ToolCallID: common.Ptr("func-123")}).
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

				expectations := []persistCallExpectation{
					{
						Role:          domain.ChatRole_User,
						Content:       "Call the same tool repeatedly",
						ID:            &userMsgID,
						ToolCallsLen:  0,
						HasToolCallID: false,
					},
				}
				for range 5 {
					expectations = append(expectations,
						persistCallExpectation{
							Role:          domain.ChatRole_Assistant,
							Content:       "",
							ToolCallsLen:  1,
							HasToolCallID: false,
						},
						persistCallExpectation{
							Role:          domain.ChatRole_Tool,
							Content:       "same result",
							ToolCallsLen:  0,
							HasToolCallID: true,
						},
					)
				}
				expectations = append(expectations, persistCallExpectation{
					Role:          domain.ChatRole_Assistant,
					Content:       "Sorry, I could not process your request. Please try again.",
					ID:            &assistantMsgID,
					ToolCallsLen:  0,
					HasToolCallID: false,
				})
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, expectations)
			},
			expectErr:       false,
			expectedContent: "calling fetch_todos...\ncalling fetch_todos...\ncalling fetch_todos...\ncalling fetch_todos...\ncalling fetch_todos...\nSorry, I could not process your request. Please try again.\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testStreamChatImpl(t, tt)
		})
	}
}
