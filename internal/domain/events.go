package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

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
	ChatRole       ChatRole
	ChatMessageID  uuid.UUID
	ConversationID string
}

// EventPublisher defines the interface for publishing events.
type EventPublisher interface {
	PublishEvent(ctx context.Context, event OutboxEvent) error
}
