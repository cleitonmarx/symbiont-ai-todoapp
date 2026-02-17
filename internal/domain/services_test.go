package domain

import (
	"testing"

	"github.com/google/uuid"
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

func TestShouldHandleConversationTitleGenerationEvent(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	tests := map[string]struct {
		event      ChatMessageEvent
		wantHandle bool
		wantErr    string
	}{
		"invalid-event-type": {
			event: ChatMessageEvent{
				Type:           EventType_TODO_CREATED,
				ConversationID: conversationID,
				ChatRole:       ChatRole_Assistant,
			},
			wantHandle: false,
			wantErr:    "invalid event type for conversation title generation",
		},
		"empty-conversation-id": {
			event: ChatMessageEvent{
				Type:     EventType_CHAT_MESSAGE_SENT,
				ChatRole: ChatRole_Assistant,
			},
			wantHandle: false,
			wantErr:    "conversation id cannot be empty",
		},
		"non-assistant-event": {
			event: ChatMessageEvent{
				Type:           EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatRole:       ChatRole_User,
			},
			wantHandle: false,
		},
		"assistant-event": {
			event: ChatMessageEvent{
				Type:           EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatRole:       ChatRole_Assistant,
			},
			wantHandle: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotHandle, err := ShouldHandleConversationTitleGenerationEvent(tt.event)
			assert.Equal(t, tt.wantHandle, gotHandle)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestShouldHandleConversationSummaryGenerationEvent(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	tests := map[string]struct {
		event      ChatMessageEvent
		wantHandle bool
		wantErr    string
	}{
		"invalid-event-type": {
			event: ChatMessageEvent{
				Type:           EventType_TODO_CREATED,
				ConversationID: conversationID,
			},
			wantHandle: false,
			wantErr:    "invalid event type for chat summary",
		},
		"empty-conversation-id": {
			event: ChatMessageEvent{
				Type: EventType_CHAT_MESSAGE_SENT,
			},
			wantHandle: false,
			wantErr:    "conversation id cannot be empty",
		},
		"valid-chat-message-event": {
			event: ChatMessageEvent{
				Type:           EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
			},
			wantHandle: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotHandle, err := ShouldHandleConversationSummaryGenerationEvent(tt.event)
			assert.Equal(t, tt.wantHandle, gotHandle)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestDetermineConversationSummaryGenerationDecision(t *testing.T) {
	policy := ConversationSummaryGenerationPolicy{
		TriggerMessageCount: 10,
		TriggerTokenCount:   2000,
	}

	tools := map[string]struct{}{
		"create_todo": {},
		"update_todo": {},
	}

	tests := map[string]struct {
		messages []ChatMessage
		hasMore  bool
		tools    map[string]struct{}
		want     ConversationSummaryGenerationDecision
	}{
		"triggered-by-state-changing-tool-success": {
			messages: func() []ChatMessage {
				toolCallID := "tool-1"
				return []ChatMessage{
					{
						ChatRole: ChatRole_Assistant,
						ToolCalls: []LLMStreamEventToolCall{
							{ID: toolCallID, Function: "create_todo"},
						},
					},
					{
						ChatRole:     ChatRole_Tool,
						ToolCallID:   &toolCallID,
						MessageState: ChatMessageState_Completed,
					},
				}
			}(),
			tools: tools,
			want: ConversationSummaryGenerationDecision{
				ShouldGenerate: true,
				Reason:         ConversationSummaryGenerationReason_StateChangingToolSuccess,
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
			tools: tools,
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
			tools:   tools,
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
			tools: tools,
			want: ConversationSummaryGenerationDecision{
				ShouldGenerate: true,
				Reason:         ConversationSummaryGenerationReason_TokenCountThreshold,
				MessageCount:   1,
				TotalTokens:    2001,
			},
		},
		"does-not-trigger-for-non-state-changing-tool": {
			messages: func() []ChatMessage {
				toolCallID := "tool-2"
				return []ChatMessage{
					{
						ChatRole: ChatRole_Assistant,
						ToolCalls: []LLMStreamEventToolCall{
							{ID: toolCallID, Function: "search_todo"},
						},
					},
					{
						ChatRole:     ChatRole_Tool,
						ToolCallID:   &toolCallID,
						MessageState: ChatMessageState_Completed,
					},
				}
			}(),
			tools: tools,
			want: ConversationSummaryGenerationDecision{
				ShouldGenerate: false,
				Reason:         ConversationSummaryGenerationReason_None,
				MessageCount:   2,
				TotalTokens:    0,
			},
		},
		"does-not-trigger-with-empty-tools-config": {
			messages: func() []ChatMessage {
				toolCallID := "tool-3"
				return []ChatMessage{
					{
						ChatRole: ChatRole_Assistant,
						ToolCalls: []LLMStreamEventToolCall{
							{ID: toolCallID, Function: "create_todo"},
						},
					},
					{
						ChatRole:     ChatRole_Tool,
						ToolCallID:   &toolCallID,
						MessageState: ChatMessageState_Completed,
					},
				}
			}(),
			tools: map[string]struct{}{},
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
			got := DetermineConversationSummaryGenerationDecision(tt.messages, tt.hasMore, policy, tt.tools)
			assert.Equal(t, tt.want, got)
		})
	}
}
