package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TodoEventType string

const (
	// TodoEventType_TODO_CREATED represents the event when a todo item is created.
	TodoEventType_TODO_CREATED TodoEventType = "TODO.CREATED"
	// TodoEventType_TODO_UPDATED represents the event when a todo item is updated.
	TodoEventType_TODO_UPDATED TodoEventType = "TODO.UPDATED"
	// TodoEventType_TODO_DELETED represents the event when a todo item is deleted.
	TodoEventType_TODO_DELETED TodoEventType = "TODO.DELETED"
)

// TodoEvent represents a domain event in the system.
type TodoEvent struct {
	Type      TodoEventType
	TodoID    uuid.UUID
	CreatedAt time.Time
}

// TodoEventPublisher defines the interface for publishing todo events.
type TodoEventPublisher interface {
	PublishEvent(ctx context.Context, event OutboxEvent) error
}
