package chat

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type actionTestRenderer struct {
	rendered assistant.Message
	ok       bool
}

func (r actionTestRenderer) Render(_ assistant.ActionCall, _ assistant.Message) (assistant.Message, bool) {
	return r.rendered, r.ok
}

func TestStreamChatImpl_Execute_ActionCases(t *testing.T) {
	t.Parallel()

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
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				conversationRepo *assistant.MockConversationRepository,
				timeProvider *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
				actionRegistry *assistant.MockActionRegistry,
				skillRegistry *assistant.MockSkillRegistry,
				uow *transaction.MockUnitOfWork,
				outbox *outbox.MockRepository,
			) {
				skillRegistry.EXPECT().
					ListRelevant(mock.Anything, mock.Anything).
					Return([]assistant.SkillDefinition{}).
					Once()

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(assistant.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()
				actionRegistry.EXPECT().
					StatusMessage("list_todos").
					Return("calling list_todos")

				actionRegistry.EXPECT().
					Execute(
						mock.Anything,
						assistant.ActionCall{
							ID:    "func-123",
							Name:  "list_todos",
							Input: "{\"page\": 1, \"page_size\": 5, \"search_term\": \"searchTerm\"}",
							Text:  "calling list_todos",
						},
						mock.MatchedBy(func(msgs []assistant.Message) bool {
							return len(msgs) > 0 && msgs[len(msgs)-1].Content == "Call an action"
						}),
					).
					Return(assistant.Message{Role: assistant.ChatRole_Tool, ActionCallID: common.Ptr("func-123")})

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				expectNowCalls(timeProvider, fixedTime, 6)

				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(
						actionFunctionCallback(userMsgID, assistantMsgID, fixedTime),
					)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Call an action",
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
							msg := "calling list_todos"
							return &msg
						}(),
					},
					{
						Role:            assistant.ChatRole_Tool,
						Content:         "",
						ActionCallsLen:  0,
						HasActionCallID: true,
						ActionExecuted:  common.Ptr(true),
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "Action called successfully.",
						ID:              &assistantMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "",
		},
		"success-with-renderer-bypasses-follow-up-runturn": {
			userMessage: "Rename my todo",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				conversationRepo *assistant.MockConversationRepository,
				timeProvider *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
				actionRegistry *assistant.MockActionRegistry,
				skillRegistry *assistant.MockSkillRegistry,
				uow *transaction.MockUnitOfWork,
				outbox *outbox.MockRepository,
			) {
				skillRegistry.EXPECT().
					ListRelevant(mock.Anything, mock.Anything).
					Return([]assistant.SkillDefinition{}).
					Once()

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(assistant.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()
				actionRegistry.EXPECT().
					StatusMessage("update_todos").
					Return("updating todos...\n")
				actionRegistry.EXPECT().
					Execute(
						mock.Anything,
						assistant.ActionCall{
							ID:    "func-123",
							Name:  "update_todos",
							Input: "{\"todos\":[{\"id\":\"1\",\"title\":\"Updated\"}]}",
							Text:  "updating todos...\n",
						},
						mock.MatchedBy(func(msgs []assistant.Message) bool {
							return len(msgs) > 0 && msgs[len(msgs)-1].Content == "Rename my todo"
						}),
					).
					Return(assistant.Message{
						Role:         assistant.ChatRole_Tool,
						ActionCallID: common.Ptr("func-123"),
						Content:      "todos[1]{id,title,due_date,status}\n1,Updated,2026-01-25,OPEN",
					}).
					Once()
				actionRegistry.EXPECT().
					GetRenderer("update_todos").
					Return(actionTestRenderer{
						rendered: assistant.Message{
							Role:    assistant.ChatRole_Assistant,
							Content: "**Updated** (Due: Jan 25, 2026) - OPEN.",
						},
						ok: true,
					}, true).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				expectNowCalls(timeProvider, fixedTime, 6)

				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
						if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						return onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
							ID:    "func-123",
							Name:  "update_todos",
							Input: `{"todos":[{"id":"1","title":"Updated"}]}`,
						})
					}).
					Once()

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Rename my todo",
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
							msg := "updating todos...\n"
							return &msg
						}(),
					},
					{
						Role:            assistant.ChatRole_Tool,
						Content:         "todos[1]{id,title,due_date,status}\n1,Updated,2026-01-25,OPEN",
						ActionCallsLen:  0,
						HasActionCallID: true,
						ActionExecuted:  common.Ptr(true),
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "**Updated** (Due: Jan 25, 2026) - OPEN.",
						ID:              &assistantMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "updating todos...\n**Updated** (Due: Jan 25, 2026) - OPEN.",
		},
		"action-message-marked-as-failed-when-content-has-error": {
			userMessage: "Call failing action",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				conversationRepo *assistant.MockConversationRepository,
				timeProvider *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
				actionRegistry *assistant.MockActionRegistry,
				skillRegistry *assistant.MockSkillRegistry,
				uow *transaction.MockUnitOfWork,
				outbox *outbox.MockRepository,
			) {
				skillRegistry.EXPECT().
					ListRelevant(mock.Anything, mock.Anything).
					Return([]assistant.SkillDefinition{}).
					Once()

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(assistant.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()
				actionRegistry.EXPECT().
					StatusMessage("failing_action").
					Return("calling failing_action...\n")

				actionRegistry.EXPECT().
					Execute(
						mock.Anything,
						assistant.ActionCall{
							ID:    "func-error",
							Name:  "failing_action",
							Input: "{\"input\":\"x\"}",
							Text:  "calling failing_action...\n",
						},
						mock.MatchedBy(func(msgs []assistant.Message) bool {
							return len(msgs) > 0 && msgs[len(msgs)-1].Content == "Call failing action"
						}),
					).
					Return(assistant.Message{Role: assistant.ChatRole_Tool, ActionCallID: common.Ptr("func-error"), Content: "error: failing_action unavailable"})

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				expectNowCalls(timeProvider, fixedTime, 6)

				callCount := 0
				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
						if callCount == 0 {
							callCount++
							if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
								UserMessageID:      userMsgID,
								AssistantMessageID: assistantMsgID,
							}); err != nil {
								return err
							}
							return onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
								ID:    "func-error",
								Name:  "failing_action",
								Input: "{\"input\":\"x\"}",
							})
						}

						if err := onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "I could not complete that action call."}); err != nil {
							return err
						}
						return onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Times(2)

				actionErr := "error: failing_action unavailable"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Call failing action",
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
							msg := "calling failing_action...\n"
							return &msg
						}(),
					},
					{
						Role:            assistant.ChatRole_Tool,
						MessageState:    assistant.ChatMessageState_Failed,
						ErrorMessage:    &actionErr,
						Content:         actionErr,
						ActionCallsLen:  0,
						HasActionCallID: true,
						ActionExecuted:  common.Ptr(true),
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "I could not complete that action call.",
						ID:              &assistantMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "calling failing_action...\nI could not complete that action call.",
		},
		"onEvent-action-call-error": {
			userMessage: "Call action",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				conversationRepo *assistant.MockConversationRepository,
				timeProvider *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
				actionRegistry *assistant.MockActionRegistry,
				skillRegistry *assistant.MockSkillRegistry,
				uow *transaction.MockUnitOfWork,
				outbox *outbox.MockRepository,
			) {
				skillRegistry.EXPECT().
					ListRelevant(mock.Anything, mock.Anything).
					Return([]assistant.SkillDefinition{}).
					Once()

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(assistant.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()
				actionRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n")

				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
						if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						return onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
							ID:    "func-1",
							Name:  "fetch_todos",
							Input: `{"page": 1}`,
						})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Call action",
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
							msg := "calling fetch_todos...\n"
							return &msg
						}(),
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "",
						ID:              &assistantMsgID,
						MessageState:    assistant.ChatMessageState_Failed,
						ErrorMessage:    &onEventErr,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				})
			},
			expectErr:      true,
			onEventErrType: assistant.EventType_ActionStarted,
		},
		"onEvent-action-call-finished-error": {
			userMessage: "Call action",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				conversationRepo *assistant.MockConversationRepository,
				timeProvider *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
				actionRegistry *assistant.MockActionRegistry,
				skillRegistry *assistant.MockSkillRegistry,
				uow *transaction.MockUnitOfWork,
				outbox *outbox.MockRepository,
			) {
				skillRegistry.EXPECT().
					ListRelevant(mock.Anything, mock.Anything).
					Return([]assistant.SkillDefinition{}).
					Once()

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(assistant.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()
				actionRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n")

				actionRegistry.EXPECT().
					Execute(mock.Anything, mock.Anything, mock.Anything).
					Return(assistant.Message{Role: assistant.ChatRole_Tool, ActionCallID: common.Ptr("func-1"), Content: "action result"}).
					Once()

				expectNowCalls(timeProvider, fixedTime, 5)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
						if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						return onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
							ID:    "func-1",
							Name:  "fetch_todos",
							Input: `{"page": 1}`,
						})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Call action",
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
							msg := "calling fetch_todos...\n"
							return &msg
						}(),
					},
					{
						Role:            assistant.ChatRole_Tool,
						Content:         "action result",
						ActionCallsLen:  0,
						HasActionCallID: true,
						ActionExecuted:  common.Ptr(true),
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "",
						ID:              &assistantMsgID,
						MessageState:    assistant.ChatMessageState_Failed,
						ErrorMessage:    &onEventErr,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				})
			},
			expectErr:      true,
			onEventErrType: assistant.EventType_ActionCompleted,
		},
		"max-action-cycles-exceeded": {
			userMessage: "Keep calling actions",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				conversationRepo *assistant.MockConversationRepository,
				timeProvider *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
				actionRegistry *assistant.MockActionRegistry,
				skillRegistry *assistant.MockSkillRegistry,
				uow *transaction.MockUnitOfWork,
				outbox *outbox.MockRepository,
			) {
				skillRegistry.EXPECT().
					ListRelevant(mock.Anything, mock.Anything).
					Return([]assistant.SkillDefinition{}).
					Once()

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(assistant.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()
				expectNowCalls(timeProvider, fixedTime, 18)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				actionRegistry.EXPECT().
					StatusMessage(mock.Anything).
					Return("calling action...\n").
					Times(7)

				actionRegistry.EXPECT().
					Execute(mock.Anything, mock.Anything, mock.Anything).
					Return(assistant.Message{Role: assistant.ChatRole_Tool, Content: "action result", ActionCallID: common.Ptr("func-123")}).
					Times(7)

				callCount := 0
				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
						if callCount == 0 {
							if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
								UserMessageID:      userMsgID,
								AssistantMessageID: assistantMsgID,
							}); err != nil {
								return err
							}
						}

						callCount++
						return onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
							ID:    fmt.Sprintf("func-%d", callCount),
							Name:  "fetch_todos",
							Input: fmt.Sprintf(`{"page": %d}`, callCount),
						})
					}).
					Times(8)

				expectations := []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Keep calling actions",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				}
				for i := 0; i < 7; i++ {
					expectations = append(expectations,
						persistCallExpectation{
							Role:            assistant.ChatRole_Assistant,
							Content:         "",
							ActionCallsLen:  1,
							HasActionCallID: false,
						},
						persistCallExpectation{
							Role:            assistant.ChatRole_Tool,
							Content:         "action result",
							ActionCallsLen:  0,
							HasActionCallID: true,
						},
					)
				}
				expectations = append(expectations, persistCallExpectation{
					Role:            assistant.ChatRole_Assistant,
					Content:         "Sorry, I could not process your request. Please try again.",
					ID:              &assistantMsgID,
					ActionCallsLen:  0,
					HasActionCallID: false,
				})
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, expectations)
			},
			expectErr:       false,
			expectedContent: "calling action...\ncalling action...\ncalling action...\ncalling action...\ncalling action...\ncalling action...\ncalling action...\nSorry, I could not process your request. Please try again.\n",
		},
		"repeated-action-call-loop": {
			userMessage: "Call the same action repeatedly",
			model:       "test-model",
			options: []StreamChatOption{
				WithConversationID(conversationID),
			},
			setExpectations: func(
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				conversationRepo *assistant.MockConversationRepository,
				timeProvider *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
				actionRegistry *assistant.MockActionRegistry,
				skillRegistry *assistant.MockSkillRegistry,
				uow *transaction.MockUnitOfWork,
				outbox *outbox.MockRepository,
			) {
				skillRegistry.EXPECT().
					ListRelevant(mock.Anything, mock.Anything).
					Return([]assistant.SkillDefinition{}).
					Once()

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(assistant.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()
				expectNowCalls(timeProvider, fixedTime, 14)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				actionRegistry.EXPECT().
					StatusMessage("fetch_todos").
					Return("calling fetch_todos...\n").
					Times(5)

				actionRegistry.EXPECT().
					Execute(mock.Anything, mock.Anything, mock.Anything).
					Return(assistant.Message{Role: assistant.ChatRole_Tool, Content: "same result", ActionCallID: common.Ptr("func-123")}).
					Times(5)

				callCount := 0
				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
						if callCount == 0 {
							if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
								UserMessageID:      userMsgID,
								AssistantMessageID: assistantMsgID,
							}); err != nil {
								return err
							}
						}

						callCount++
						return onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
							ID:    fmt.Sprintf("func-%d", callCount),
							Name:  "fetch_todos",
							Input: `{"page": 1}`,
						})
					}).
					Times(6)

				expectations := []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Call the same action repeatedly",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				}
				for range 5 {
					expectations = append(expectations,
						persistCallExpectation{
							Role:            assistant.ChatRole_Assistant,
							Content:         "",
							ActionCallsLen:  1,
							HasActionCallID: false,
						},
						persistCallExpectation{
							Role:            assistant.ChatRole_Tool,
							Content:         "same result",
							ActionCallsLen:  0,
							HasActionCallID: true,
						},
					)
				}
				expectations = append(expectations, persistCallExpectation{
					Role:            assistant.ChatRole_Assistant,
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
