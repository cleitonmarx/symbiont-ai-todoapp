package outbox

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
)

// EventType identifies the kind of domain event carried by outbox messages.
type EventType string

const (
	// EventType_TODO_CREATED represents the event when a todo item is created.
	EventType_TODO_CREATED EventType = "TODO.CREATED"
	// EventType_TODO_UPDATED represents the event when a todo item is updated.
	EventType_TODO_UPDATED EventType = "TODO.UPDATED"
	// EventType_TODO_DELETED represents the event when a todo item is deleted.
	EventType_TODO_DELETED EventType = "TODO.DELETED"
	// EventType_CHAT_MESSAGE_SENT represents the event when a chat message is sent.
	EventType_CHAT_MESSAGE_SENT EventType = "CHAT_MESSAGE.SENT"
	// EventType_ACTION_APPROVAL_DECIDED represents a human approval decision for an assistant action call.
	EventType_ACTION_APPROVAL_DECIDED EventType = "ACTION_APPROVAL.DECIDED"
)

// TodoEvent represents a domain event in the system.
type TodoEvent struct {
	Type      EventType
	TodoID    uuid.UUID
	CreatedAt time.Time
}

// ChatMessageEvent represents a domain event for chat messages in the system.
type ChatMessageEvent struct {
	Type           EventType
	ChatRole       assistant.ChatRole
	ChatMessageID  uuid.UUID
	ConversationID uuid.UUID
	CreatedAt      time.Time
}

// EventPublisher defines the interface for publishing events.
type EventPublisher interface {
	PublishEvent(ctx context.Context, event Event) error
}
