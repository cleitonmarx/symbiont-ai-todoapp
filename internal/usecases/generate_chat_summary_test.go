package usecases

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGenerateChatSummaryImpl_Execute(t *testing.T) {
	conversationID := "global"
	chatMessageID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	checkpointID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedTime := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		event           domain.ChatMessageEvent
		model           string
		setExpectations func(
			*testing.T,
			*domain.MockChatMessageRepository,
			*domain.MockConversationSummaryRepository,
			*domain.MockCurrentTimeProvider,
			*domain.MockLLMClient,
		)
		expectedErr error
	}{
		"invalid-event-type": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_TODO_CREATED,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			expectedErr: domain.NewValidationErr("invalid event type for chat summary"),
		},
		"empty-conversation-id": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:          domain.EventType_CHAT_MESSAGE_SENT,
				ChatMessageID: chatMessageID,
			},
			expectedErr: domain.NewValidationErr("conversation id cannot be empty"),
		},
		"get-summary-error": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, errors.New("summary db error")).
					Once()
			},
			expectedErr: fmt.Errorf("failed to get conversation summary: %w", errors.New("summary db error")),
		},
		"list-chat-messages-error": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					RunAndReturn(func(ctx context.Context, limit int, options ...domain.ListChatMessagesOption) ([]domain.ChatMessage, bool, error) {
						assert.Equal(t, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, limit)
						resolved := domain.ListChatMessagesOptions{}
						for _, option := range options {
							option(&resolved)
						}
						assert.Equal(t, conversationID, resolved.ConversationID)
						assert.Nil(t, resolved.AfterMessageID)
						return nil, false, errors.New("chat db error")
					}).
					Once()
			},
			expectedErr: fmt.Errorf("failed to list chat messages: %w", errors.New("chat db error")),
		},
		"no-unsummarized-messages-noop": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					Return([]domain.ChatMessage{}, false, nil).
					Once()
			},
			expectedErr: nil,
		},
		"below-threshold-noop": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					Return([]domain.ChatMessage{
						{
							ID:             chatMessageID,
							ConversationID: conversationID,
							ChatRole:       domain.ChatRole_Assistant,
							Content:        "short text",
							MessageState:   domain.ChatMessageState_Completed,
						},
					}, false, nil).
					Once()
			},
			expectedErr: nil,
		},
		"trigger-by-message-count-success": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				existingSummaryID := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{
						ID:                      existingSummaryID,
						ConversationID:          conversationID,
						CurrentStateSummary:     "Current: planning tasks",
						LastSummarizedMessageID: &checkpointID,
					}, true, nil).
					Once()

				messages := make([]domain.ChatMessage, 0, CHAT_SUMMARY_TRIGGER_MESSAGES)
				lastMessageID := uuid.Nil
				for idx := range CHAT_SUMMARY_TRIGGER_MESSAGES {
					lastMessageID = uuid.New()
					role := domain.ChatRole_User
					if idx%2 == 1 {
						role = domain.ChatRole_Assistant
					}
					messages = append(messages, domain.ChatMessage{
						ID:             lastMessageID,
						ConversationID: conversationID,
						ChatRole:       role,
						Content:        "message",
						MessageState:   domain.ChatMessageState_Completed,
					})
				}
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					RunAndReturn(func(ctx context.Context, limit int, options ...domain.ListChatMessagesOption) ([]domain.ChatMessage, bool, error) {
						resolved := domain.ListChatMessagesOptions{}
						for _, option := range options {
							option(&resolved)
						}
						assert.Equal(t, conversationID, resolved.ConversationID)
						if assert.NotNil(t, resolved.AfterMessageID) {
							assert.Equal(t, checkpointID, *resolved.AfterMessageID)
						}
						return messages, false, nil
					}).
					Once()

				llmClient.EXPECT().
					Chat(mock.Anything, mock.MatchedBy(func(req domain.LLMChatRequest) bool {
						return assert.Equal(t, "summary-model", req.Model) &&
							assert.Equal(t, common.Ptr(CHAT_SUMMARY_MAX_TOKENS), req.MaxTokens) &&
							assert.Equal(t, common.Ptr(CHAT_SUMMARY_FREQUENCY_PENALTY), req.FrequencyPenalty)
					})).
					Return(domain.LLMChatResponse{
						Content: "Current State:\n- User asked to organize tasks.",
						Usage: domain.LLMUsage{
							PromptTokens:     9,
							CompletionTokens: 4,
						},
					}, nil).
					Once()

				timeProvider.EXPECT().Now().Return(fixedTime).Once()

				summaryRepo.EXPECT().
					StoreConversationSummary(mock.Anything, mock.MatchedBy(func(summary domain.ConversationSummary) bool {
						if summary.ID != existingSummaryID {
							return false
						}
						if summary.LastSummarizedMessageID == nil {
							return false
						}
						return *summary.LastSummarizedMessageID == lastMessageID
					})).
					Return(nil).
					Once()
			},
			expectedErr: nil,
		},
		"trigger-by-token-threshold-success": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					Return([]domain.ChatMessage{
						{
							ID:             chatMessageID,
							ConversationID: conversationID,
							ChatRole:       domain.ChatRole_Assistant,
							Content:        "short",
							MessageState:   domain.ChatMessageState_Completed,
							TotalTokens:    CHAT_SUMMARY_TRIGGER_TOKENS + 1,
						},
					}, false, nil).
					Once()
				llmClient.EXPECT().
					Chat(mock.Anything, mock.Anything).
					Return(domain.LLMChatResponse{Content: "summary"}, nil).
					Once()
				timeProvider.EXPECT().Now().Return(fixedTime).Once()
				summaryRepo.EXPECT().
					StoreConversationSummary(mock.Anything, mock.Anything).
					Return(nil).
					Once()
			},
			expectedErr: nil,
		},
		"trigger-by-state-changing-tool-success": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_Tool,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				toolCallID := "tool-call-1"
				assistantMsgID := uuid.MustParse("623e4567-e89b-12d3-a456-426614174005")
				toolMsgID := uuid.MustParse("723e4567-e89b-12d3-a456-426614174006")
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					Return([]domain.ChatMessage{
						{
							ID:             assistantMsgID,
							ConversationID: conversationID,
							ChatRole:       domain.ChatRole_Assistant,
							ToolCalls: []domain.LLMStreamEventToolCall{
								{
									ID:       toolCallID,
									Function: "create_todo",
								},
							},
							MessageState: domain.ChatMessageState_Completed,
						},
						{
							ID:             toolMsgID,
							ConversationID: conversationID,
							ChatRole:       domain.ChatRole_Tool,
							ToolCallID:     &toolCallID,
							Content:        `{"message":"ok"}`,
							MessageState:   domain.ChatMessageState_Completed,
						},
					}, false, nil).
					Once()
				llmClient.EXPECT().
					Chat(mock.Anything, mock.Anything).
					Return(domain.LLMChatResponse{Content: "summary"}, nil).
					Once()
				timeProvider.EXPECT().Now().Return(fixedTime).Once()
				summaryRepo.EXPECT().
					StoreConversationSummary(mock.Anything, mock.Anything).
					Return(nil).
					Once()
			},
			expectedErr: nil,
		},
		"empty-llm-content-noop": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					Return([]domain.ChatMessage{
						{
							ID:             chatMessageID,
							ConversationID: conversationID,
							ChatRole:       domain.ChatRole_Assistant,
							Content:        "hello",
							MessageState:   domain.ChatMessageState_Completed,
						},
					}, true, nil).
					Once()
				llmClient.EXPECT().
					Chat(mock.Anything, mock.Anything).
					Return(domain.LLMChatResponse{
						Content: "",
						Usage: domain.LLMUsage{
							PromptTokens:     120,
							CompletionTokens: 512,
						},
					}, nil).
					Once()
			},
			expectedErr: nil,
		},
		"llm-error": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					Return([]domain.ChatMessage{
						{
							ID:             chatMessageID,
							ConversationID: conversationID,
							ChatRole:       domain.ChatRole_Assistant,
							Content:        "hello",
							MessageState:   domain.ChatMessageState_Completed,
						},
					}, true, nil).
					Once()
				llmClient.EXPECT().
					Chat(mock.Anything, mock.Anything).
					Return(domain.LLMChatResponse{}, errors.New("llm error")).
					Once()
			},
			expectedErr: fmt.Errorf("failed to generate chat summary: %w", errors.New("llm error")),
		},
		"store-error": {
			model: "summary-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			setExpectations: func(
				t *testing.T,
				chatRepo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					Return([]domain.ChatMessage{
						{
							ID:             chatMessageID,
							ConversationID: conversationID,
							ChatRole:       domain.ChatRole_Assistant,
							Content:        "hello",
							MessageState:   domain.ChatMessageState_Completed,
						},
					}, true, nil).
					Once()
				llmClient.EXPECT().
					Chat(mock.Anything, mock.Anything).
					Return(domain.LLMChatResponse{Content: "summary"}, nil).
					Once()
				timeProvider.EXPECT().Now().Return(fixedTime).Once()
				summaryRepo.EXPECT().
					StoreConversationSummary(mock.Anything, mock.Anything).
					Return(errors.New("store error")).
					Once()
			},
			expectedErr: fmt.Errorf("failed to store conversation summary: %w", errors.New("store error")),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			chatRepo := domain.NewMockChatMessageRepository(t)
			summaryRepo := domain.NewMockConversationSummaryRepository(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			llmClient := domain.NewMockLLMClient(t)

			if tt.setExpectations != nil {
				tt.setExpectations(t, chatRepo, summaryRepo, timeProvider, llmClient)
			}

			uc := NewGenerateChatSummaryImpl(
				chatRepo,
				summaryRepo,
				timeProvider,
				llmClient,
				tt.model,
				nil,
			)

			gotErr := uc.Execute(context.Background(), tt.event)
			assert.Equal(t, tt.expectedErr, gotErr)
		})
	}
}

func TestInitGenerateChatSummary_Initialize(t *testing.T) {
	i := InitGenerateChatSummary{}

	ctx, err := i.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	uc, err := depend.Resolve[GenerateChatSummary]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)
}
