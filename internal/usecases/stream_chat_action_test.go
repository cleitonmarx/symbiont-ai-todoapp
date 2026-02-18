package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestStreamChatImpl_Execute_ActionCases(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userMsgID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	assistantMsgID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]streamChatTestTableEntry{
		"success-with-action-call": {
			userMessage: "Call an action",
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

				actionRegistry.EXPECT().
					StatusMessage("list_todos").
					Return("calling list_todos")

				actionRegistry.EXPECT().
					Execute(
						mock.Anything,
						domain.AssistantActionCall{
							ID:    "func-123",
							Name:  "list_todos",
							Input: "{\"page\": 1, \"page_size\": 5, \"search_term\": \"searchTerm\"}",
							Text:  "calling list_todos",
						},
						mock.MatchedBy(func(msgs []domain.AssistantMessage) bool {
							return len(msgs) > 0 && msgs[len(msgs)-1].Content == "Call an action"
						}),
					).
					Return(domain.AssistantMessage{Role: domain.ChatRole_Tool, ActionCallID: common.Ptr("func-123")})

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				expectNowCalls(timeProvider, fixedTime, 7)

				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(
						toolFunctionCallback(userMsgID, assistantMsgID, fixedTime),
					)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            domain.ChatRole_User,
						Content:         "Call an action",
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
						Role:            domain.ChatRole_Tool,
						Content:         "",
						ActionCallsLen:  0,
						HasActionCallID: true,
					},
					{
						Role:            domain.ChatRole_Assistant,
						Content:         "Tool called successfully.",
						ID:              &assistantMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
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

				actionRegistry.EXPECT().
					StatusMessage("failing_tool").
					Return("calling failing_tool...\n")

				actionRegistry.EXPECT().
					Execute(
						mock.Anything,
						domain.AssistantActionCall{
							ID:    "func-error",
							Name:  "failing_tool",
							Input: "{\"input\":\"x\"}",
							Text:  "calling failing_tool...\n",
						},
						mock.MatchedBy(func(msgs []domain.AssistantMessage) bool {
							return len(msgs) > 0 && msgs[len(msgs)-1].Content == "Call failing tool"
						}),
					).
					Return(domain.AssistantMessage{Role: domain.ChatRole_Tool, ActionCallID: common.Ptr("func-error"), Content: "error: failing_tool unavailable"})

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				expectNowCalls(timeProvider, fixedTime, 7)

				callCount := 0
				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
						if callCount == 0 {
							callCount++
							if err := onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
								UserMessageID:      userMsgID,
								AssistantMessageID: assistantMsgID,
							}); err != nil {
								return err
							}
							return onEvent(domain.AssistantEventType_ActionRequested, domain.AssistantActionCall{
								ID:    "func-error",
								Name:  "failing_tool",
								Input: "{\"input\":\"x\"}",
							})
						}

						if err := onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "I could not complete that tool call."}); err != nil {
							return err
						}
						return onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Times(2)

				toolErr := "error: failing_tool unavailable"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            domain.ChatRole_User,
						Content:         "Call failing tool",
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
						Role:            domain.ChatRole_Tool,
						MessageState:    domain.ChatMessageState_Failed,
						ErrorMessage:    &toolErr,
						Content:         toolErr,
						ActionCallsLen:  0,
						HasActionCallID: true,
					},
					{
						Role:            domain.ChatRole_Assistant,
						Content:         "I could not complete that tool call.",
						ID:              &assistantMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "calling failing_tool...\nI could not complete that tool call.",
		},
		"onEvent-action-call-error": {
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

				actionRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n")

				expectNowCalls(timeProvider, fixedTime, 5)

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
						return onEvent(domain.AssistantEventType_ActionRequested, domain.AssistantActionCall{
							ID:    "func-1",
							Name:  "fetch_todos",
							Input: `{"page": 1}`,
						})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            domain.ChatRole_User,
						Content:         "Call tool",
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
						Role:            domain.ChatRole_Assistant,
						Content:         "",
						ID:              &assistantMsgID,
						MessageState:    domain.ChatMessageState_Failed,
						ErrorMessage:    &onEventErr,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				})
			},
			expectErr:      true,
			onEventErrType: domain.AssistantEventType_ActionStarted,
		},
		"onEvent-tool-call-finished-error": {
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

				actionRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n")

				actionRegistry.EXPECT().
					Execute(mock.Anything, mock.Anything, mock.Anything).
					Return(domain.AssistantMessage{Role: domain.ChatRole_Tool, ActionCallID: common.Ptr("func-1"), Content: "tool result"}).
					Once()

				expectNowCalls(timeProvider, fixedTime, 6)

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
						return onEvent(domain.AssistantEventType_ActionRequested, domain.AssistantActionCall{
							ID:    "func-1",
							Name:  "fetch_todos",
							Input: `{"page": 1}`,
						})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            domain.ChatRole_User,
						Content:         "Call tool",
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
						Role:            domain.ChatRole_Tool,
						Content:         "tool result",
						ActionCallsLen:  0,
						HasActionCallID: true,
					},
					{
						Role:            domain.ChatRole_Assistant,
						Content:         "",
						ID:              &assistantMsgID,
						MessageState:    domain.ChatMessageState_Failed,
						ErrorMessage:    &onEventErr,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				})
			},
			expectErr:      true,
			onEventErrType: domain.AssistantEventType_ActionCompleted,
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

				expectNowCalls(timeProvider, fixedTime, 19)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				actionRegistry.EXPECT().
					StatusMessage(mock.Anything).
					Return("calling tool...\n").
					Times(7)

				actionRegistry.EXPECT().
					Execute(mock.Anything, mock.Anything, mock.Anything).
					Return(domain.AssistantMessage{Role: domain.ChatRole_Tool, Content: "tool result", ActionCallID: common.Ptr("func-123")}).
					Times(7)

				callCount := 0
				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
						if callCount == 0 {
							if err := onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
								UserMessageID:      userMsgID,
								AssistantMessageID: assistantMsgID,
							}); err != nil {
								return err
							}
						}

						callCount++
						return onEvent(domain.AssistantEventType_ActionRequested, domain.AssistantActionCall{
							ID:    fmt.Sprintf("func-%d", callCount),
							Name:  "fetch_todos",
							Input: fmt.Sprintf(`{"page": %d}`, callCount),
						})
					}).
					Times(8)

				expectations := []persistCallExpectation{
					{
						Role:            domain.ChatRole_User,
						Content:         "Keep calling tools",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				}
				for i := 0; i < 7; i++ {
					expectations = append(expectations,
						persistCallExpectation{
							Role:            domain.ChatRole_Assistant,
							Content:         "",
							ActionCallsLen:  1,
							HasActionCallID: false,
						},
						persistCallExpectation{
							Role:            domain.ChatRole_Tool,
							Content:         "tool result",
							ActionCallsLen:  0,
							HasActionCallID: true,
						},
					)
				}
				expectations = append(expectations, persistCallExpectation{
					Role:            domain.ChatRole_Assistant,
					Content:         "Sorry, I could not process your request. Please try again.",
					ID:              &assistantMsgID,
					ActionCallsLen:  0,
					HasActionCallID: false,
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

				expectNowCalls(timeProvider, fixedTime, 15)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				actionRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n").
					Times(5)

				actionRegistry.EXPECT().
					Execute(mock.Anything, mock.Anything, mock.Anything).
					Return(domain.AssistantMessage{Role: domain.ChatRole_Tool, Content: "same result", ActionCallID: common.Ptr("func-123")}).
					Times(5)

				callCount := 0
				assistant.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
						if callCount == 0 {
							if err := onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
								UserMessageID:      userMsgID,
								AssistantMessageID: assistantMsgID,
							}); err != nil {
								return err
							}
						}

						callCount++
						return onEvent(domain.AssistantEventType_ActionRequested, domain.AssistantActionCall{
							ID:    fmt.Sprintf("func-%d", callCount),
							Name:  "fetch_todos",
							Input: `{"page": 1}`,
						})
					}).
					Times(6)

				expectations := []persistCallExpectation{
					{
						Role:            domain.ChatRole_User,
						Content:         "Call the same tool repeatedly",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				}
				for range 5 {
					expectations = append(expectations,
						persistCallExpectation{
							Role:            domain.ChatRole_Assistant,
							Content:         "",
							ActionCallsLen:  1,
							HasActionCallID: false,
						},
						persistCallExpectation{
							Role:            domain.ChatRole_Tool,
							Content:         "same result",
							ActionCallsLen:  0,
							HasActionCallID: true,
						},
					)
				}
				expectations = append(expectations, persistCallExpectation{
					Role:            domain.ChatRole_Assistant,
					Content:         "Sorry, I could not process your request. Please try again.",
					ID:              &assistantMsgID,
					ActionCallsLen:  0,
					HasActionCallID: false,
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

func Test(t *testing.T) {
	b, _ := json.Marshal(domain.AssistantActionCall{
		ID:    "func-123",
		Name:  "list_todos",
		Input: `{"page": 1, "page_size": 5, "search_term": "searchTerm"}`,
		Text:  "calling list_todos",
	})

	fmt.Println(string(b))
}
