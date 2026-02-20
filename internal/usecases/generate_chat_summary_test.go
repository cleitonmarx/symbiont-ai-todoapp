package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
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
			*domain.MockAssistant,
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
				assistant *domain.MockAssistant,
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
				assistant *domain.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					RunAndReturn(func(ctx context.Context, conversationID uuid.UUID, page int, limit int, options ...domain.ListChatMessagesOption) ([]domain.ChatMessage, bool, error) {
						assert.Equal(t, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, limit)
						resolved := domain.ListChatMessagesParams{}
						for _, option := range options {
							option(&resolved)
						}
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
				assistant *domain.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
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
				assistant *domain.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
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
				assistant *domain.MockAssistant,
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
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					RunAndReturn(func(ctx context.Context, conversationID uuid.UUID, page int, limit int, options ...domain.ListChatMessagesOption) ([]domain.ChatMessage, bool, error) {
						resolved := domain.ListChatMessagesParams{}
						for _, option := range options {
							option(&resolved)
						}
						if assert.NotNil(t, resolved.AfterMessageID) {
							assert.Equal(t, checkpointID, *resolved.AfterMessageID)
						}
						return messages, false, nil
					}).
					Once()

				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.MatchedBy(func(req domain.AssistantTurnRequest) bool {
						return assert.Equal(t, "summary-model", req.Model) &&
							assert.Equal(t, common.Ptr(CHAT_SUMMARY_MAX_TOKENS), req.MaxTokens) &&
							assert.Equal(t, common.Ptr(CHAT_SUMMARY_FREQUENCY_PENALTY), req.FrequencyPenalty)
					})).
					Return(domain.AssistantTurnResponse{
						Content: "Current State:\n- User asked to organize tasks.",
						Usage: domain.AssistantUsage{
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
				assistant *domain.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
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
				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{Content: "summary"}, nil).
					Once()
				timeProvider.EXPECT().Now().Return(fixedTime).Once()
				summaryRepo.EXPECT().
					StoreConversationSummary(mock.Anything, mock.Anything).
					Return(nil).
					Once()
			},
			expectedErr: nil,
		},
		"trigger-by-state-changing-action-success": {
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
				assistant *domain.MockAssistant,
			) {
				actionCallID := "action-call-1"
				assistantMsgID := uuid.MustParse("623e4567-e89b-12d3-a456-426614174005")
				actionMsgID := uuid.MustParse("723e4567-e89b-12d3-a456-426614174006")
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
					Return([]domain.ChatMessage{
						{
							ID:             assistantMsgID,
							ConversationID: conversationID,
							ChatRole:       domain.ChatRole_Assistant,
							ActionCalls: []domain.AssistantActionCall{
								{
									ID:   actionCallID,
									Name: "create_todo",
								},
							},
							MessageState: domain.ChatMessageState_Completed,
						},
						{
							ID:             actionMsgID,
							ConversationID: conversationID,
							ChatRole:       domain.ChatRole_Tool,
							ActionCallID:   &actionCallID,
							Content:        `{"message":"ok"}`,
							MessageState:   domain.ChatMessageState_Completed,
						},
					}, false, nil).
					Once()
				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{Content: "summary"}, nil).
					Once()
				timeProvider.EXPECT().Now().Return(fixedTime).Once()
				summaryRepo.EXPECT().
					StoreConversationSummary(mock.Anything, mock.MatchedBy(func(summary domain.ConversationSummary) bool {
						return strings.Contains(summary.CurrentStateSummary, "recent_action_calls: create_todo")
					})).
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
				assistant *domain.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
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
				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{
						Content: "",
						Usage: domain.AssistantUsage{
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
				assistant *domain.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
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
				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{}, errors.New("llm error")).
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
				assistant *domain.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, mock.Anything).
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
				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{Content: "summary"}, nil).
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
			assistant := domain.NewMockAssistant(t)

			if tt.setExpectations != nil {
				tt.setExpectations(t, chatRepo, summaryRepo, timeProvider, assistant)
			}

			uc := NewGenerateChatSummaryImpl(
				chatRepo,
				summaryRepo,
				timeProvider,
				assistant,
				tt.model,
				nil,
			)

			gotErr := uc.Execute(context.Background(), tt.event)
			assert.Equal(t, tt.expectedErr, gotErr)
		})
	}
}

func TestMergeRecentActionCallsIntoSummary(t *testing.T) {
	tests := map[string]struct {
		previousSummary string
		newSummary      string
		messages        []domain.ChatMessage
		expectedValue   string
		expectedCount   int
	}{
		"adds-recent-action-calls-field-when-missing": {
			previousSummary: "current_intent: plan tasks\nlast_action: none\noutput_format: concise text",
			newSummary:      "current_intent: plan tasks\nactive_view: none\nuser_nuances: none\ntasks: none\nlast_action: summarized\noutput_format: concise text",
			messages: []domain.ChatMessage{
				{
					ChatRole: domain.ChatRole_Assistant,
					ActionCalls: []domain.AssistantActionCall{
						{Name: "fetch_todos"},
						{Name: "set_ui_filters"},
					},
				},
			},
			expectedValue: "fetch_todos; set_ui_filters",
			expectedCount: 1,
		},
		"caps-recent-action-calls-at-five": {
			previousSummary: "recent_action_calls: call1; call2; call3; call4; call5; call6; call7; call8; call9",
			newSummary:      "current_intent: x\nactive_view: none\nuser_nuances: none\ntasks: none\nlast_action: none\noutput_format: concise text",
			messages: []domain.ChatMessage{
				{
					ChatRole: domain.ChatRole_Assistant,
					ActionCalls: []domain.AssistantActionCall{
						{Name: "call10"},
						{Name: "call11"},
					},
				},
			},
			expectedValue: "call7; call8; call9; call10; call11",
			expectedCount: 1,
		},
		"replaces-existing-field-in-new-summary": {
			previousSummary: "current_intent: x",
			newSummary:      "current_intent: x\nrecent_action_calls: stale\nlast_action: none\noutput_format: concise text",
			messages: []domain.ChatMessage{
				{
					ChatRole: domain.ChatRole_Assistant,
					ActionCalls: []domain.AssistantActionCall{
						{Name: "create_todo"},
					},
				},
			},
			expectedValue: "create_todo",
			expectedCount: 1,
		},
		"does-not-duplicate-when-last-action-precedes-existing-field": {
			previousSummary: "recent_action_calls: set_ui_filters; fetch_todos",
			newSummary:      "current_intent: Generate summary of April tasks with due dates and statuses\nactive_view: Todos filtered by due date (April 2026) and status (OPEN), sorted by due date ascending\nuser_nuances: Requested overview summary of tasks, prefers concise format with due dates and statuses\ntasks: Plan team meeting agenda|O|2026-04-01; Organize digital files|O|2026-04-02; Read industry news|O|2026-04-03; Prepare for book club|O|2026-04-04\nlast_action: get_tasks -> Success -> Retrieved 10 tasks for April 2026\nrecent_action_calls: stale\noutput_format: summary with counters",
			messages: []domain.ChatMessage{
				{
					ChatRole: domain.ChatRole_Assistant,
					ActionCalls: []domain.AssistantActionCall{
						{Name: "set_ui_filters"},
						{Name: "fetch_todos"},
					},
				},
			},
			expectedValue: "set_ui_filters; fetch_todos; set_ui_filters; fetch_todos",
			expectedCount: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := mergeRecentActionCallsIntoSummary(tt.previousSummary, tt.newSummary, tt.messages)
			value, ok := findSummaryFieldValue(got, SUMMARY_RECENT_ACTION_CALLS_FIELD)
			assert.True(t, ok)
			assert.Equal(t, tt.expectedValue, value)
			assert.Equal(t, tt.expectedCount, strings.Count(got, SUMMARY_RECENT_ACTION_CALLS_FIELD+":"))
		})
	}
}

func TestNormalizeConversationSummary(t *testing.T) {
	tests := map[string]struct {
		previous string
		candidate string
		assertions func(t *testing.T, got string)
	}{
		"fills-missing-fields-from-previous-and-defaults": {
			previous: strings.Join([]string{
				"current_intent: plan sprint tasks",
				"active_view: project AI open tasks",
				"user_nuances: concise answers",
				"tasks: task one|O|2026-02-20",
				"last_action: fetch_todos -> success",
				"recent_action_calls: fetch_todos; set_ui_filters",
				"open_loops: fix wrong due date",
				"output_format: list",
			}, "\n"),
			candidate: strings.Join([]string{
				"current_intent: update due date for backend task",
				"last_action: update_todo_due_date -> success",
			}, "\n"),
			assertions: func(t *testing.T, got string) {
				assert.Equal(t, len(summaryOrderedFields), len(strings.Split(got, "\n")))
				assert.Contains(t, got, "current_intent: update due date for backend task")
				assert.Contains(t, got, "active_view: project AI open tasks")
				assert.Contains(t, got, "open_loops: fix wrong due date")
				assert.Contains(t, got, "output_format: list")
			},
		},
		"handles-malformed-candidate-with-compact-defaults": {
			previous:  "No current state.",
			candidate: "Current state: user wants help",
			assertions: func(t *testing.T, got string) {
				assert.Equal(t, len(summaryOrderedFields), len(strings.Split(got, "\n")))
				assert.Contains(t, got, "current_intent: none")
				assert.Contains(t, got, "recent_action_calls: none")
				assert.Contains(t, got, "open_loops: none")
				assert.Contains(t, got, "output_format: concise text")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := normalizeConversationSummary(tt.previous, tt.candidate)
			tt.assertions(t, got)
		})
	}
}

func TestFormatMessageForSummary(t *testing.T) {
	tests := map[string]struct {
		message       domain.ChatMessage
		expectContain []string
		maxContentLen int
	}{
		"tool-message-is-compacted-and-keeps-action-signal": {
			message: domain.ChatMessage{
				ChatRole:     domain.ChatRole_Tool,
				MessageState: domain.ChatMessageState_Completed,
				Content:      strings.Repeat("long tool output ", 30),
				ActionCalls: []domain.AssistantActionCall{
					{Name: "fetch_todos"},
					{Name: "set_ui_filters"},
				},
			},
			expectContain: []string{
				"- role: tool",
				"action_calls: fetch_todos; set_ui_filters",
				"action_success: false",
			},
			maxContentLen: MAX_SUMMARY_TOOL_CONTENT_CHARS + 3,
		},
		"empty-content-is-normalized-to-none": {
			message: domain.ChatMessage{
				ChatRole:     domain.ChatRole_User,
				MessageState: domain.ChatMessageState_Completed,
				Content:      "   ",
			},
			expectContain: []string{
				"- role: user",
				"content: none",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := formatMessageForSummary(tt.message)
			for _, expected := range tt.expectContain {
				assert.Contains(t, got, expected)
			}
			if tt.maxContentLen > 0 {
				lines := strings.Split(got, "\n")
				contentLine := ""
				for _, line := range lines {
					if strings.HasPrefix(line, "  content: ") {
						contentLine = strings.TrimPrefix(line, "  content: ")
						break
					}
				}
				if assert.NotEmpty(t, contentLine) {
					assert.LessOrEqual(t, len([]rune(contentLine)), tt.maxContentLen)
				}
			}
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
