package chat

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConversationCompactorImpl_Compact(t *testing.T) {
	t.Parallel()
	var CHAT_SUMMARY_TRIGGER_TOKENS = 8000

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	chatMessageID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
	largeContextMessage := strings.Repeat("a", CHAT_SUMMARY_TRIGGER_TOKENS*4)

	tests := map[string]struct {
		model           string
		conversationID  uuid.UUID
		setExpectations func(
			*assistant.MockChatMessageRepository,
			*assistant.MockConversationSummaryRepository,
			*core.MockCurrentTimeProvider,
			*assistant.MockAssistant,
		)
		expectedErr string
	}{
		"empty-conversation-id": {
			model:          "summary-model",
			conversationID: uuid.Nil,
			expectedErr:    "conversation id cannot be empty",
		},
		"get-summary-error": {
			model:          "summary-model",
			conversationID: conversationID,
			setExpectations: func(
				_ *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				_ *core.MockCurrentTimeProvider,
				_ *assistant.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(assistant.ConversationSummary{}, false, errors.New("summary db error")).
					Once()
			},
			expectedErr: "failed to get conversation summary: summary db error",
		},
		"list-chat-messages-error": {
			model:          "summary-model",
			conversationID: conversationID,
			setExpectations: func(
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				_ *core.MockCurrentTimeProvider,
				_ *assistant.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(assistant.ConversationSummary{}, false, nil).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 0).
					Return(nil, false, errors.New("chat db error")).
					Once()
			},
			expectedErr: "failed to list chat messages: chat db error",
		},
		"no-unsummarized-messages-noop": {
			model:          "summary-model",
			conversationID: conversationID,
			setExpectations: func(
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				_ *core.MockCurrentTimeProvider,
				_ *assistant.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(assistant.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 0).
					Return([]assistant.ChatMessage{}, false, nil).
					Once()
			},
			expectedErr: "",
		},
		"success": {
			model:          "summary-model",
			conversationID: conversationID,
			setExpectations: func(
				chatRepo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				timeProvider *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
			) {
				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(assistant.ConversationSummary{}, false, nil).
					Once()
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 0).
					Return([]assistant.ChatMessage{
						{
							ID:                    chatMessageID,
							ConversationID:        conversationID,
							ChatRole:              assistant.ChatRole_Assistant,
							Content:               largeContextMessage,
							MessageState:          assistant.ChatMessageState_Completed,
							ContextTokensEstimate: CHAT_SUMMARY_TRIGGER_TOKENS + 1,
						},
					}, false, nil).
					Once()
				assist.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(assistant.TurnResponse{Content: "memory: compacted\ncarry: pending confirmation"}, nil).
					Once()
				timeProvider.EXPECT().Now().Return(fixedTime).Once()
				summaryRepo.EXPECT().
					StoreConversationSummary(mock.Anything, mock.MatchedBy(func(summary assistant.ConversationSummary) bool {
						return summary.ConversationID == conversationID &&
							summary.LastSummarizedMessageID != nil &&
							*summary.LastSummarizedMessageID == chatMessageID &&
							summary.CurrentStateSummary == "memory: compacted\ncarry: pending confirmation" &&
							summary.UpdatedAt.Equal(fixedTime)
					})).
					Return(nil).
					Once()
			},
			expectedErr: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			chatRepo := assistant.NewMockChatMessageRepository(t)
			summaryRepo := assistant.NewMockConversationSummaryRepository(t)
			timeProvider := core.NewMockCurrentTimeProvider(t)
			assistantClient := assistant.NewMockAssistant(t)

			if tt.setExpectations != nil {
				tt.setExpectations(chatRepo, summaryRepo, timeProvider, assistantClient)
			}

			uc := NewConversationCompactorImpl(
				chatRepo,
				summaryRepo,
				timeProvider,
				assistantClient,
				tt.model,
			)

			gotErr := uc.Compact(t.Context(), tt.conversationID)
			if tt.expectedErr == "" {
				assert.NoError(t, gotErr)
				return
			}
			require.EqualError(t, gotErr, tt.expectedErr)
		})
	}
}

func TestConversationCompactorImpl_EvaluateConversationCompaction(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	checkpointID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")

	chatRepo := assistant.NewMockChatMessageRepository(t)
	summaryRepo := assistant.NewMockConversationSummaryRepository(t)
	timeProvider := core.NewMockCurrentTimeProvider(t)
	assistantClient := assistant.NewMockAssistant(t)

	summaryRepo.EXPECT().
		GetConversationSummary(mock.Anything, conversationID).
		Return(assistant.ConversationSummary{
			ConversationID:          conversationID,
			LastSummarizedMessageID: &checkpointID,
		}, true, nil).
		Once()

	chatRepo.EXPECT().
		ListChatMessages(
			mock.Anything,
			conversationID,
			1,
			0,
			mock.MatchedBy(func(options []assistant.ListChatMessagesOption) bool {
				if len(options) != 1 {
					return false
				}
				params := assistant.ListChatMessagesParams{}
				options[0](&params)
				return params.AfterMessageID != nil && *params.AfterMessageID == checkpointID
			}),
		).
		Return([]assistant.ChatMessage{
			{ContextTokensEstimate: 4000},
			{ContextTokensEstimate: 5001},
		}, false, nil).
		Once()

	uc := NewConversationCompactorImpl(
		chatRepo,
		summaryRepo,
		timeProvider,
		assistantClient,
		"summary-model",
	)

	decision, err := uc.EvaluateConversationCompaction(t.Context(), conversationID, assistant.CompactionPolicy{
		TriggerTokenCount: 8000,
	})

	require.NoError(t, err)
	assert.True(t, decision.ShouldCompact)
	assert.Equal(t, assistant.ContextCompactionReasonTokenCountThreshold, decision.Reason)
	assert.Equal(t, 2, decision.MessageCount)
	assert.Equal(t, 9001, decision.TotalTokens)
}

func TestNormalizeConversationSummary(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		previous   string
		candidate  string
		assertions func(t *testing.T, got string)
	}{
		"keeps-compact-transcript-lines": {
			previous: "memory: existing compacted context",
			candidate: strings.Join([]string{
				"user: update due date for backend task",
				"tool: update_todo_due_date success; backend task due 2026-02-21",
				"carry: confirm if frontend task should move too",
			}, "\n"),
			assertions: func(t *testing.T, got string) {
				lines := strings.Split(got, "\n")
				assert.Len(t, lines, 3)
				assert.Equal(t, "user: update due date for backend task", lines[0])
				assert.Equal(t, "tool: update_todo_due_date success; backend task due 2026-02-21", lines[1])
				assert.Equal(t, "carry: confirm if frontend task should move too", lines[2])
			},
		},
		"strips-fences-bullets-and-whitespace": {
			previous:  "No current state.",
			candidate: "```text\n- memory: dinner planning for Feb 20\n- carry: confirm menu and guest count\n```",
			assertions: func(t *testing.T, got string) {
				assert.Equal(t, "text\nmemory: dinner planning for Feb 20\ncarry: confirm menu and guest count", got)
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
	t.Parallel()

	tests := map[string]struct {
		message       assistant.ChatMessage
		expectContain []string
		maxContentLen int
	}{
		"tool-message-is-compacted-and-keeps-action-signal": {
			message: assistant.ChatMessage{
				ChatRole:     assistant.ChatRole_Tool,
				MessageState: assistant.ChatMessageState_Completed,
				Content:      strings.Repeat("long tool output ", 30),
				ActionCalls: []assistant.ActionCall{
					{Name: "fetch_todos"},
					{Name: "set_ui_filters"},
				},
			},
			expectContain: []string{
				"tool: ",
				"calls=fetch_todos; set_ui_filters",
				"success=false",
			},
			maxContentLen: MAX_SUMMARY_TOOL_CONTENT_CHARS + 3,
		},
		"empty-content-is-normalized-to-none": {
			message: assistant.ChatMessage{
				ChatRole:     assistant.ChatRole_User,
				MessageState: assistant.ChatMessageState_Completed,
				Content:      "   ",
			},
			expectContain: []string{
				"user: none",
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
				prefixEnd := strings.Index(got, ": ")
				require.NotEqual(t, -1, prefixEnd)
				contentSegment := got[prefixEnd+2:]
				if separator := strings.Index(contentSegment, " | "); separator >= 0 {
					contentSegment = contentSegment[:separator]
				}
				assert.LessOrEqual(t, len([]rune(contentSegment)), tt.maxContentLen)
			}
		})
	}
}
