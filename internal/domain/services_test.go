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
