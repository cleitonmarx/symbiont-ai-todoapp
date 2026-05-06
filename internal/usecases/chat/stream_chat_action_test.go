package chat

import (
	"context"
	"errors"
	"fmt"
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
			fixedTime:   fixedTime,
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
						actionFunctionCallback(),
					)

			},
			persistExpectations: []persistCallExpectation{
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
			},
			expectErr:       false,
			expectedContent: "",
		},
		"success-with-renderer-continues-follow-up-runturn": {
			userMessage: "Rename my todo",
			model:       "test-model",
			fixedTime:   fixedTime,
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
						if len(req.Messages) == 0 {
							return fmt.Errorf("expected request messages")
						}
						if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{}); err != nil {
							return err
						}
						return onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
							ID:    "func-123",
							Name:  "update_todos",
							Input: `{"todos":[{"id":"1","title":"Updated"}]}`,
						})
					}).
					Once()

			},
			persistExpectations: []persistCallExpectation{
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
			},
			expectErr:       false,
			expectedContent: "updating todos...\n**Updated** (Due: Jan 25, 2026) - OPEN.",
		},
		"action-message-marked-as-failed-when-content-has-error": {
			userMessage: "Call failing action",
			model:       "test-model",
			fixedTime:   fixedTime,
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
					Return(assistant.Message{
						Role:         assistant.ChatRole_Tool,
						ActionCallID: common.Ptr("func-error"),
						Content:      "error: failing_action unavailable",
						ActionError:  common.Ptr("error: failing_action unavailable"),
					})

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
							if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{}); err != nil {
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
						return onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{})
					}).
					Times(2)

			},
			persistExpectations: func() []persistCallExpectation {
				actionErr := "error: failing_action unavailable"
				return []persistCallExpectation{
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
				}
			}(),
			expectErr:       false,
			expectedContent: "calling failing_action...\nI could not complete that action call.",
		},
		"max-action-cycles-exceeded": {
			userMessage: "Keep calling actions",
			model:       "test-model",
			fixedTime:   fixedTime,
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
							if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{}); err != nil {
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

			},
			persistExpectations: func() []persistCallExpectation {
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
				return expectations
			}(),
			expectErr:       false,
			expectedContent: "calling action...\ncalling action...\ncalling action...\ncalling action...\ncalling action...\ncalling action...\ncalling action...\nSorry, I could not process your request. Please try again.\n",
		},
		"repeated-action-call-loop": {
			userMessage: "Call the same action repeatedly",
			model:       "test-model",
			fixedTime:   fixedTime,
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
							if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{}); err != nil {
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

			},
			persistExpectations: func() []persistCallExpectation {
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
				return expectations
			}(),
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

func TestStreamChatImpl_Execute_CanceledTurnRepairsDanglingActionCall(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)
	actionCallID := "func-123"

	chatRepo := assistant.NewMockChatMessageRepository(t)
	summaryRepo := assistant.NewMockConversationSummaryRepository(t)
	conversationRepo := assistant.NewMockConversationRepository(t)
	timeProvider := core.NewMockCurrentTimeProvider(t)
	assist := assistant.NewMockAssistant(t)
	actionRegistry := assistant.NewMockActionRegistry(t)
	skillRegistry := assistant.NewMockSkillRegistry(t)
	uow := transaction.NewMockUnitOfWork(t)
	outboxRepo := outbox.NewMockRepository(t)

	conversation := assistant.Conversation{ID: conversationID, CreatedAt: fixedTime.Add(-time.Minute)}

	skillRegistry.EXPECT().
		ListRelevant(mock.Anything, mock.Anything).
		Return([]assistant.SkillDefinition{}).
		Once()

	conversationRepo.EXPECT().
		GetConversation(mock.Anything, conversationID).
		Return(conversation, true, nil).
		Once()

	summaryRepo.EXPECT().
		GetConversationSummary(mock.Anything, conversationID).
		Return(assistant.ConversationSummary{}, false, nil).
		Once()

	chatRepo.EXPECT().
		ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
		Return([]assistant.ChatMessage{}, false, nil).
		Once()

	expectNowCalls(timeProvider, fixedTime, 2)

	actionRegistry.EXPECT().
		StatusMessage("list_todos").
		Return("calling list_todos").
		Once()

	assistantActionCallMessageID := uuid.Nil
	userMessageID := uuid.Nil
	turnID := uuid.Nil

	createScope1 := transaction.NewMockScope(t)
	createScope2 := transaction.NewMockScope(t)
	repairScope := transaction.NewMockScope(t)

	uow.EXPECT().
		Execute(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
			return fn(ctx, createScope1)
		}).
		Once()
	uow.EXPECT().
		Execute(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
			return fn(ctx, createScope2)
		}).
		Once()
	uow.EXPECT().
		Execute(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
			return fn(ctx, repairScope)
		}).
		Once()

	createScope1.EXPECT().ChatMessage().Return(chatRepo).Once()
	createScope1.EXPECT().Outbox().Return(outboxRepo).Once()
	createScope1.EXPECT().Conversation().Return(conversationRepo).Once()

	createScope2.EXPECT().ChatMessage().Return(chatRepo).Once()
	createScope2.EXPECT().Outbox().Return(outboxRepo).Once()
	createScope2.EXPECT().Conversation().Return(conversationRepo).Once()

	repairScope.EXPECT().ChatMessage().Return(chatRepo).Twice()
	repairScope.EXPECT().Conversation().Return(conversationRepo).Twice()

	chatRepo.EXPECT().
		CreateChatMessages(mock.Anything, mock.MatchedBy(func(msgs []assistant.ChatMessage) bool {
			if len(msgs) != 1 {
				return false
			}
			msg := msgs[0]
			if msg.ChatRole != assistant.ChatRole_User || msg.Content != "Call an action" {
				return false
			}
			userMessageID = msg.ID
			return true
		})).
		Return(nil).
		Once()

	outboxRepo.EXPECT().
		CreateChatEvent(mock.Anything, mock.MatchedBy(func(event outbox.ChatMessageEvent) bool {
			return event.ChatRole == assistant.ChatRole_User && event.ChatMessageID == userMessageID
		})).
		Return(nil).
		Once()

	conversationRepo.EXPECT().
		UpdateConversation(mock.Anything, mock.MatchedBy(func(conv assistant.Conversation) bool {
			return conv.ID == conversationID && conv.LastMessageAt != nil && conv.LastMessageAt.Equal(fixedTime) && conv.UpdatedAt.Equal(fixedTime)
		})).
		Return(nil).
		Once()

	chatRepo.EXPECT().
		CreateChatMessages(mock.Anything, mock.MatchedBy(func(msgs []assistant.ChatMessage) bool {
			if len(msgs) != 1 {
				return false
			}
			msg := msgs[0]
			if msg.ChatRole != assistant.ChatRole_Assistant || len(msg.ActionCalls) != 1 || msg.ActionCalls[0].ID != actionCallID {
				return false
			}
			assistantActionCallMessageID = msg.ID
			return true
		})).
		Return(nil).
		Once()

	outboxRepo.EXPECT().
		CreateChatEvent(mock.Anything, mock.MatchedBy(func(event outbox.ChatMessageEvent) bool {
			return event.ChatRole == assistant.ChatRole_Assistant && event.ChatMessageID == assistantActionCallMessageID
		})).
		Return(nil).
		Once()

	conversationRepo.EXPECT().
		UpdateConversation(mock.Anything, mock.MatchedBy(func(conv assistant.Conversation) bool {
			return conv.ID == conversationID && conv.LastMessageAt != nil && conv.LastMessageAt.Equal(fixedTime) && conv.UpdatedAt.Equal(fixedTime)
		})).
		Return(nil).
		Once()

	assist.EXPECT().
		RunTurn(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
			return onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
				ID:    actionCallID,
				Name:  "list_todos",
				Input: `{"page": 1}`,
			})
		}).
		Once()

	chatRepo.EXPECT().
		ListChatMessages(mock.Anything, conversationID, 1, 0).
		RunAndReturn(func(ctx context.Context, conversationID uuid.UUID, page int, pageSize int, options ...assistant.ListChatMessagesOption) ([]assistant.ChatMessage, bool, error) {
			return []assistant.ChatMessage{
				{
					ID:             userMessageID,
					ConversationID: conversationID,
					TurnID:         turnID,
					ChatRole:       assistant.ChatRole_User,
					Content:        "Call an action",
					CreatedAt:      fixedTime,
				},
				{
					ID:             assistantActionCallMessageID,
					ConversationID: conversationID,
					TurnID:         turnID,
					ChatRole:       assistant.ChatRole_Assistant,
					ActionCalls: []assistant.ActionCall{
						{ID: actionCallID, Name: "list_todos"},
					},
					CreatedAt: fixedTime,
				},
			}, false, nil
		}).
		Once()

	chatRepo.EXPECT().
		DeleteChatMessages(mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
			return len(ids) == 1 && ids[0] == assistantActionCallMessageID
		})).
		Return(nil).
		Once()

	conversationRepo.EXPECT().
		GetConversation(mock.Anything, conversationID).
		Return(assistant.Conversation{
			ID:          conversationID,
			Title:       "Fresh title",
			TitleSource: assistant.ConversationTitleSource_LLM,
			CreatedAt:   fixedTime.Add(-time.Minute),
			UpdatedAt:   fixedTime,
		}, true, nil).
		Once()

	conversationRepo.EXPECT().
		UpdateConversation(mock.Anything, mock.MatchedBy(func(conv assistant.Conversation) bool {
			return conv.ID == conversationID &&
				conv.Title == "Fresh title" &&
				conv.TitleSource == assistant.ConversationTitleSource_LLM &&
				conv.LastMessageAt != nil &&
				conv.LastMessageAt.Equal(fixedTime) &&
				conv.UpdatedAt.Equal(fixedTime)
		})).
		Return(nil).
		Once()

	useCase := newTestStreamChatUseCase(
		log.New(io.Discard, "", 0),
		chatRepo,
		summaryRepo,
		nil,
		conversationRepo,
		timeProvider,
		nil,
		assist,
		actionRegistry,
		skillRegistry,
		nil,
		uow,
		7,
		8000,
		DEFAULT_CONTEXT_COMPACTION_TIMEOUT,
	)

	err := useCase.Execute(t.Context(), "Call an action", "test-model", func(_ context.Context, eventType assistant.EventType, data any) error {
		if eventType == assistant.EventType_TurnStarted {
			turnID = data.(assistant.TurnStarted).TurnID
		}
		if eventType == assistant.EventType_ActionStarted {
			return context.Canceled
		}
		return nil
	}, WithConversationID(conversationID))

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if turnID == uuid.Nil {
		t.Fatal("expected a turn id to be emitted before cancellation")
	}
}
