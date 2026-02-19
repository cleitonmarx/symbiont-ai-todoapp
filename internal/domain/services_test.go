package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAutoConversationTitle(t *testing.T) {
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

func TestDetermineConversationSummaryGenerationDecision(t *testing.T) {
	policy := ConversationSummaryGenerationPolicy{
		TriggerMessageCount: 10,
		TriggerTokenCount:   2000,
	}

	actions := map[string]struct{}{
		"create_todo": {},
		"update_todo": {},
	}

	tests := map[string]struct {
		messages []ChatMessage
		hasMore  bool
		actions  map[string]struct{}
		want     ConversationSummaryGenerationDecision
	}{
		"triggered-by-state-changing-action-success": {
			messages: func() []ChatMessage {
				actionCallID := "action-1"
				return []ChatMessage{
					{
						ChatRole: ChatRole_Assistant,
						ActionCalls: []AssistantActionCall{
							{ID: actionCallID, Name: "create_todo"},
						},
					},
					{
						ChatRole:     ChatRole_Tool,
						ActionCallID: &actionCallID,
						MessageState: ChatMessageState_Completed,
					},
				}
			}(),
			actions: actions,
			want: ConversationSummaryGenerationDecision{
				ShouldGenerate: true,
				Reason:         ConversationSummaryGenerationReason_StateChangingActionSuccess,
				MessageCount:   2,
				TotalTokens:    0,
			},
		},
		"triggered-by-message-count-threshold": {
			messages: []ChatMessage{
				{ChatRole: ChatRole_User},
				{ChatRole: ChatRole_Assistant},
				{ChatRole: ChatRole_User},
				{ChatRole: ChatRole_Assistant},
				{ChatRole: ChatRole_User},
				{ChatRole: ChatRole_Assistant},
				{ChatRole: ChatRole_User},
				{ChatRole: ChatRole_Assistant},
				{ChatRole: ChatRole_User},
				{ChatRole: ChatRole_Assistant},
			},
			actions: actions,
			want: ConversationSummaryGenerationDecision{
				ShouldGenerate: true,
				Reason:         ConversationSummaryGenerationReason_MessageCountThreshold,
				MessageCount:   10,
				TotalTokens:    0,
			},
		},
		"triggered-by-has-more": {
			messages: []ChatMessage{
				{ChatRole: ChatRole_User},
			},
			hasMore: true,
			actions: actions,
			want: ConversationSummaryGenerationDecision{
				ShouldGenerate: true,
				Reason:         ConversationSummaryGenerationReason_MessageCountThreshold,
				MessageCount:   1,
				TotalTokens:    0,
			},
		},
		"triggered-by-token-count-threshold": {
			messages: []ChatMessage{
				{ChatRole: ChatRole_Assistant, TotalTokens: 2001},
			},
			actions: actions,
			want: ConversationSummaryGenerationDecision{
				ShouldGenerate: true,
				Reason:         ConversationSummaryGenerationReason_TokenCountThreshold,
				MessageCount:   1,
				TotalTokens:    2001,
			},
		},
		"does-not-trigger-for-non-state-changing-action": {
			messages: func() []ChatMessage {
				actionCallID := "action-2"
				return []ChatMessage{
					{
						ChatRole: ChatRole_Assistant,
						ActionCalls: []AssistantActionCall{
							{ID: actionCallID, Name: "search_todo"},
						},
					},
					{
						ChatRole:     ChatRole_Tool,
						ActionCallID: &actionCallID,
						MessageState: ChatMessageState_Completed,
					},
				}
			}(),
			actions: actions,
			want: ConversationSummaryGenerationDecision{
				ShouldGenerate: false,
				Reason:         ConversationSummaryGenerationReason_None,
				MessageCount:   2,
				TotalTokens:    0,
			},
		},
		"does-not-trigger-with-empty-actions-config": {
			messages: func() []ChatMessage {
				actionCallID := "action-3"
				return []ChatMessage{
					{
						ChatRole: ChatRole_Assistant,
						ActionCalls: []AssistantActionCall{
							{ID: actionCallID, Name: "create_todo"},
						},
					},
					{
						ChatRole:     ChatRole_Tool,
						ActionCallID: &actionCallID,
						MessageState: ChatMessageState_Completed,
					},
				}
			}(),
			actions: map[string]struct{}{},
			want: ConversationSummaryGenerationDecision{
				ShouldGenerate: false,
				Reason:         ConversationSummaryGenerationReason_None,
				MessageCount:   2,
				TotalTokens:    0,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := DetermineConversationSummaryGenerationDecision(tt.messages, tt.hasMore, policy, tt.actions)
			assert.Equal(t, tt.want, got)
		})
	}
}
