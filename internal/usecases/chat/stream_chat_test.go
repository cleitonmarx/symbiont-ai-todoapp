package chat

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStreamChatImpl_Execute(t *testing.T) {
	t.Parallel()

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

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(assistant.ConversationSummary{
						ID:                  uuid.MustParse("323e4567-e89b-12d3-a456-426614174002"),
						ConversationID:      conversationID,
						CurrentStateSummary: "Current intent: organize todos",
					}, true, nil).
					Once()
				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{
						{
							ID:        uuid.New(),
							ChatRole:  assistant.ChatRole_User,
							Content:   "Previous message",
							CreatedAt: fixedTime.Add(-time.Minute),
						},
					}, false, nil).
					Once()

				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) {
						foundSummaryContext := false
						for _, msg := range req.Messages {
							if msg.Role == assistant.ChatRole_System && strings.Contains(msg.Content, "Current intent: organize todos") {
								foundSummaryContext = true
								break
							}
						}
						assert.True(t, foundSummaryContext)

						// Simulate events
						_ = onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
						_ = onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "I'm "})
						_ = onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "doing "})
						_ = onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "great!"})
						_ = onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
							Usage: assistant.Usage{
								PromptTokens:     promptTokens,
								CompletionTokens: completionTokens,
								TotalTokens:      totalTokens,
							},
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Hello, how are you?",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
					{
						Role:             assistant.ChatRole_Assistant,
						Content:          "I'm doing great!",
						ID:               &assistantMsgID,
						PromptTokens:     &promptTokens,
						CompletionTokens: &completionTokens,
						TotalTokens:      &totalTokens,
						ActionCallsLen:   0,
						HasActionCallID:  false,
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
					CreateConversation(
						mock.Anything,
						"Hello, how are you?",
						assistant.ConversationTitleSource_Auto,
					).
					Return(assistant.Conversation{
						ID:          conversationID,
						Title:       "Hello, how are you?",
						TitleSource: assistant.ConversationTitleSource_Auto,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					}, nil)

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(assistant.ConversationSummary{
						ID:                  uuid.MustParse("323e4567-e89b-12d3-a456-426614174002"),
						ConversationID:      conversationID,
						CurrentStateSummary: "Current intent: organize todos",
					}, true, nil).
					Once()
				expectNowCalls(timeProvider, fixedTime, 4)

				// history: empty
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{
						{
							ID:        uuid.New(),
							ChatRole:  assistant.ChatRole_User,
							Content:   "Previous message",
							CreatedAt: fixedTime.Add(-time.Minute),
						},
					}, false, nil).
					Once()

				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) {
						foundSummaryContext := false
						for _, msg := range req.Messages {
							if msg.Role == assistant.ChatRole_System && strings.Contains(msg.Content, "Current intent: organize todos") {
								foundSummaryContext = true
								break
							}
						}
						assert.True(t, foundSummaryContext)

						// Simulate events
						_ = onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
						_ = onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "I'm "})
						_ = onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "doing "})
						_ = onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "great!"})
						_ = onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
							Usage: assistant.Usage{
								PromptTokens:     promptTokens,
								CompletionTokens: completionTokens,
								TotalTokens:      totalTokens,
							},
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Hello, how are you?",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
					{
						Role:             assistant.ChatRole_Assistant,
						Content:          "I'm doing great!",
						ID:               &assistantMsgID,
						PromptTokens:     &promptTokens,
						CompletionTokens: &completionTokens,
						TotalTokens:      &totalTokens,
						ActionCallsLen:   0,
						HasActionCallID:  false,
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
				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) {
						_ = onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
						_ = onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Test",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "Sorry, I could not process your request. Please try again.",
						ID:              &assistantMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
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

				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(assistant.Conversation{
						ID: conversationID,
					}, true, nil).
					Once()

				expectNowCalls(timeProvider, fixedTime, 1)

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
				expectNowCalls(timeProvider, fixedTime, 3)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
						return onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Test",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
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
			onEventErrType: assistant.EventType_TurnStarted,
		},
		"onEvent-delta-error": {
			userMessage: "Test",
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
				expectNowCalls(timeProvider, fixedTime, 3)

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
						return onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "Hi"})
					})

				onEventErr := "onEvent error"
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Test",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
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
			onEventErrType: assistant.EventType_MessageDelta,
		},
		"llm-chatstream-error": {
			userMessage: "Test",
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
				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				callCount := 0
				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
						if callCount == 0 {
							callCount++
							return errors.New("llm error")
						}

						assert.Empty(t, req.AvailableActions)
						assert.NotEmpty(t, req.Messages)
						lastMsg := req.Messages[len(req.Messages)-1]
						assert.Equal(t, assistant.ChatRole_System, lastMsg.Role)
						assert.Contains(t, lastMsg.Content, "The previous assistant turn failed due to an internal processing issue")

						if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
						}); err != nil {
							return err
						}
						if err := onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{
							Text: "I hit an internal error while processing your request. Please retry with a smaller scope.",
						}); err != nil {
							return err
						}
						return onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Times(2)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Test",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "I hit an internal error while processing your request. Please retry with a smaller scope.",
						ID:              &assistantMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
				})
			},
			expectErr:       false,
			expectedContent: "I hit an internal error while processing your request. Please retry with a smaller scope.",
		},
		"chatstream-without-meta-persists-user-after-loop": {
			userMessage: "No meta",
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
				expectNowCalls(timeProvider, fixedTime, 4)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()

				assist.EXPECT().
					RunTurn(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) {
						_ = onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "Hello from model"})
						_ = onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
							AssistantMessageID: "",
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "No meta",
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "Hello from model",
						ActionCallsLen:  0,
						HasActionCallID: false,
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
				expectNowCalls(timeProvider, fixedTime, 3)

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
						if err := onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "OK"}); err != nil {
							return err
						}
						return onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					})

				dbErr := errors.New("db error")
				dbErrText := dbErr.Error()
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Test",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
						CreateErr:       dbErr,
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "",
						ID:              &assistantMsgID,
						MessageState:    assistant.ChatMessageState_Failed,
						ErrorMessage:    &dbErrText,
						ActionCallsLen:  0,
						HasActionCallID: false,
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
				expectNowCalls(timeProvider, fixedTime, 3)

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
						if err := onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "OK"}); err != nil {
							return err
						}
						return onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					})

				dbErr := errors.New("db error")
				expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
					{
						Role:            assistant.ChatRole_User,
						Content:         "Test",
						ID:              &userMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
					},
					{
						Role:            assistant.ChatRole_Assistant,
						Content:         "OK",
						ID:              &assistantMsgID,
						ActionCallsLen:  0,
						HasActionCallID: false,
						CreateErr:       dbErr,
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

func TestStreamChatImpl_Execute_PersistsSelectedSkillsAndEmitsTurnMetadata(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userMsgID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	assistantMsgID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)
	selectedDefinitions := []assistant.SkillDefinition{
		{
			Name:   "update_todos",
			Source: "skills/update_todos.md",
			Tools:  []string{"fetch_todos", "update_todos"},
		},
	}
	expectedSkills := []assistant.SelectedSkill{
		assistant.NewSelectedSkill(selectedDefinitions[0]),
	}

	chatRepo := assistant.NewMockChatMessageRepository(t)
	summaryRepo := assistant.NewMockConversationSummaryRepository(t)
	conversationRepo := assistant.NewMockConversationRepository(t)
	timeProvider := core.NewMockCurrentTimeProvider(t)
	assist := assistant.NewMockAssistant(t)
	actionRegistry := assistant.NewMockActionRegistry(t)
	skillRegistry := assistant.NewMockSkillRegistry(t)
	uow := transaction.NewMockUnitOfWork(t)
	outbox := outbox.NewMockRepository(t)

	actionRegistry.EXPECT().
		GetRenderer(mock.Anything).
		Return(nil, false).
		Maybe()

	skillRegistry.EXPECT().
		ListRelevant(mock.Anything, mock.Anything).
		Return(selectedDefinitions).
		Once()

	actionRegistry.EXPECT().
		GetDefinition("fetch_todos").
		Return(assistant.ActionDefinition{Name: "fetch_todos"}, true).
		Once()
	actionRegistry.EXPECT().
		GetDefinition("update_todos").
		Return(assistant.ActionDefinition{Name: "update_todos"}, true).
		Once()

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

	expectNowCalls(timeProvider, fixedTime, 4)

	assist.EXPECT().
		RunTurn(mock.Anything, mock.Anything, mock.Anything).
		Run(func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) {
			assert.Len(t, req.AvailableActions, 2)
			_ = onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
				UserMessageID:      userMsgID,
				AssistantMessageID: assistantMsgID,
			})
			_ = onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "Done."})
			_ = onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
				AssistantMessageID: assistantMsgID.String(),
				CompletedAt:        fixedTime.Format(time.RFC3339),
			})
		}).
		Return(nil)

	expectPersistSequence(t, chatRepo, conversationRepo, uow, outbox, fixedTime, []persistCallExpectation{
		{
			Role:            assistant.ChatRole_User,
			Content:         "Update my todos",
			ID:              &userMsgID,
			ActionCallsLen:  0,
			HasActionCallID: false,
		},
		{
			Role:            assistant.ChatRole_Assistant,
			Content:         "Done.",
			ID:              &assistantMsgID,
			ActionCallsLen:  0,
			HasActionCallID: false,
			SelectedSkills:  expectedSkills,
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
		nil,
		uow,
		"test-embedding-model",
		7,
	)

	var turnStarted assistant.TurnStarted
	err := useCase.Execute(t.Context(), "Update my todos", "test-model", func(_ context.Context, eventType assistant.EventType, data any) error {
		if eventType == assistant.EventType_TurnStarted {
			turnStarted = data.(assistant.TurnStarted)
		}
		return nil
	}, WithConversationID(conversationID))

	assert.NoError(t, err)
	assert.Equal(t, conversationID, turnStarted.ConversationID)
	assert.Equal(t, userMsgID, turnStarted.UserMessageID)
	assert.Equal(t, assistantMsgID, turnStarted.AssistantMessageID)
	assert.NotEqual(t, uuid.Nil, turnStarted.TurnID)
	assert.Equal(t, expectedSkills, turnStarted.SelectedSkills)
}

// Verify that the StreamChat use case is registered
