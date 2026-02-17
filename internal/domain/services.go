package domain

import (
	"strings"

	"github.com/google/uuid"
)

// GenerateAutoConversationTitle generates a conversation title based on the user's initial message.
func GenerateAutoConversationTitle(userMessage string) string {
	// Simple heuristic: use the first 5 words of the user's message as the title, or "New Conversation" if empty.
	words := strings.Fields(userMessage)
	if len(words) == 0 {
		return "New Conversation"
	}
	if len(words) <= 5 {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:5], " ") + "..."
}

// ShouldHandleConversationTitleGenerationEvent validates whether a chat event is eligible
// for conversation title generation.
func ShouldHandleConversationTitleGenerationEvent(event ChatMessageEvent) (bool, error) {
	if event.Type != EventType_CHAT_MESSAGE_SENT {
		return false, NewValidationErr("invalid event type for conversation title generation")
	}
	if event.ConversationID == uuid.Nil {
		return false, NewValidationErr("conversation id cannot be empty")
	}
	if event.ChatRole != ChatRole_Assistant {
		return false, nil
	}
	return true, nil
}
