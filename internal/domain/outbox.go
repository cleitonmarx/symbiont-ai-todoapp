package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// OutboxStatus represents the processing lifecycle status of an outbox event.
type OutboxStatus string

const (
	// OutboxStatus_Pending indicates the event is ready to be processed.
	OutboxStatus_Pending OutboxStatus = "PENDING"
	// OutboxStatus_Failed indicates the event exceeded retries and stopped processing.
	OutboxStatus_Failed OutboxStatus = "FAILED"
	// OutboxStatus_Processed indicates the event was successfully published.
	OutboxStatus_Processed OutboxStatus = "PROCESSED"
)

// OutboxEntityType identifies the domain aggregate represented by an outbox event.
type OutboxEntityType string

const (
	// OutboxEntityType_Todo represents todo-related events.
	OutboxEntityType_Todo OutboxEntityType = "Todo"
	// OutboxEntityType_ChatMessage represents chat-message-related events.
	OutboxEntityType_ChatMessage OutboxEntityType = "ChatMessage"
)

// OutboxTopic identifies the broker topic used for publishing outbox events.
type OutboxTopic string

const (
	// OutboxTopic_Todo is the topic for todo events.
	OutboxTopic_Todo OutboxTopic = "Todo"
	// OutboxTopic_ChatMessages is the topic for chat message events.
	OutboxTopic_ChatMessages OutboxTopic = "ChatMessages"
)

// OutboxEvent represents an event stored in the outbox.
type OutboxEvent struct {
	ID          uuid.UUID
	EntityType  OutboxEntityType
	EntityID    uuid.UUID
	Topic       OutboxTopic
	EventType   EventType
	Payload     []byte
	Status      OutboxStatus
	RetryCount  int
	MaxRetries  int
	LastError   *string
	DedupeKey   *string
	AvailableAt time.Time
	ProcessedAt *time.Time
	CreatedAt   time.Time
}

// OutboxRepository defines the interface for managing outbox events.
type OutboxRepository interface {
	// CreateEvent records a new event in the outbox.
	CreateTodoEvent(ctx context.Context, event TodoEvent) error
	// CreateChatEvent records a new chat message event in the outbox.
	CreateChatEvent(ctx context.Context, event ChatMessageEvent) error
	// FetchPendingEvents retrieves a batch of pending outbox events.
	FetchPendingEvents(ctx context.Context, limit int) ([]OutboxEvent, error)
	// UpdateEvent updates the status, retry count, and last error of an outbox event.
	UpdateEvent(ctx context.Context, eventID uuid.UUID, status OutboxStatus, retryCount int, lastError string) error
	// DeleteEvent deletes an event from the outbox.
	DeleteEvent(ctx context.Context, eventID uuid.UUID) error
}
