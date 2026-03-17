package assistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAutoConversationTitle(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		userMessage string
		want        string
	}{
		"empty-message": {
			userMessage: "",
			want:        "New Conversation",
		},
		"single-word": {
			userMessage: "Hello",
			want:        "Hello",
		},
		"more-than-five-words": {
			userMessage: "Can you help me with this task please",
			want:        "Can you help me with...",
		},
		"many-words": {
			userMessage: "I need to finish the project report by tomorrow and I want it to be perfect",
			want:        "I need to finish the...",
		},
		"whitespace-only": {
			userMessage: "   ",
			want:        "New Conversation",
		},
		"multiple-spaces-between-words": {
			userMessage: "Hello    world    test",
			want:        "Hello world test",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateAutoConversationTitle(tt.userMessage)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetermineContextCompactionDecision(t *testing.T) {
	t.Parallel()

	policy := CompactionPolicy{
		TriggerTokenCount: 2000,
	}

	tests := map[string]struct {
		messages []ChatMessage
		want     CompactionDecision
	}{
		"triggered-by-token-count-threshold": {
			messages: []ChatMessage{
				{
					ChatRole:              ChatRole_Assistant,
					MessageState:          ChatMessageState_Completed,
					Content:               "short",
					TotalTokens:           1,
					ContextTokensEstimate: 2001,
				},
			},
			want: CompactionDecision{
				ShouldCompact: true,
				Reason:        ContextCompactionReasonTokenCountThreshold,
				MessageCount:  1,
				TotalTokens:   2001,
			},
		},
		"does-not-trigger-by-action-success-alone": {
			messages: func() []ChatMessage {
				actionCallID := "action-1"
				return []ChatMessage{
					{
						ChatRole:              ChatRole_Assistant,
						ContextTokensEstimate: 12,
						ActionCalls: []ActionCall{
							{ID: actionCallID, Name: "create_todo"},
						},
					},
					{
						ChatRole:              ChatRole_Tool,
						ActionCallID:          &actionCallID,
						MessageState:          ChatMessageState_Completed,
						ContextTokensEstimate: 12,
					},
				}
			}(),
			want: CompactionDecision{
				ShouldCompact: false,
				Reason:        ContextCompactionReasonNone,
				MessageCount:  2,
				TotalTokens:   24,
			},
		},
		"does-not-trigger-below-thresholds": {
			messages: []ChatMessage{
				{ChatRole: ChatRole_User, MessageState: ChatMessageState_Completed, Content: "short", ContextTokensEstimate: 5},
				{ChatRole: ChatRole_Assistant, MessageState: ChatMessageState_Completed, Content: "short", ContextTokensEstimate: 6},
			},
			want: CompactionDecision{
				ShouldCompact: false,
				Reason:        ContextCompactionReasonNone,
				MessageCount:  2,
				TotalTokens:   11,
			},
		},
		"ignores-llm-usage-total-tokens-for-thresholds": {
			messages: []ChatMessage{
				{
					ChatRole:              ChatRole_Assistant,
					MessageState:          ChatMessageState_Completed,
					Content:               "ok",
					TotalTokens:           3200,
					ContextTokensEstimate: 5,
				},
			},
			want: CompactionDecision{
				ShouldCompact: false,
				Reason:        ContextCompactionReasonNone,
				MessageCount:  1,
				TotalTokens:   5,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := DetermineContextCompactionDecision(tt.messages, policy)
			assert.Equal(t, tt.want, got)
		})
	}
}
