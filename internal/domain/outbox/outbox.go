package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Status represents the processing lifecycle status of an outbox event.
type Status string

const (
	// Status_Pending indicates the event is ready to be processed.
	Status_Pending Status = "PENDING"
	// Status_Failed indicates the event exceeded retries and stopped processing.
	Status_Failed Status = "FAILED"
	// Status_Processed indicates the event was successfully published.
	Status_Processed Status = "PROCESSED"
)

// EntityType identifies the domain aggregate represented by an outbox event.
type EntityType string

const (
	// EntityType_Todo represents todo-related events.
	EntityType_Todo EntityType = "Todo"
	// EntityType_ChatMessage represents chat-message-related events.
	EntityType_ChatMessage EntityType = "ChatMessage"
)

// Topic identifies the broker topic used for publishing outbox events.
type Topic string

const (
	// Topic_Todo is the topic for todo events.
	Topic_Todo Topic = "Todo"
	// Topic_ChatMessages is the topic for chat message events.
	Topic_ChatMessages Topic = "ChatMessages"
	// Topic_ActionApprovals is the topic for action approval decision events.
	Topic_ActionApprovals Topic = "ActionApprovals"
)

// Event represents an event stored in the outbox.
type Event struct {
	ID          uuid.UUID
	EntityType  EntityType
	EntityID    uuid.UUID
	Topic       Topic
	EventType   EventType
	Payload     []byte
	Status      Status
	RetryCount  int
	MaxRetries  int
	LastError   *string
	DedupeKey   *string
	AvailableAt time.Time
	ProcessedAt *time.Time
	CreatedAt   time.Time
}

// Repository defines the interface for managing outbox events.
type Repository interface {
	// CreateEvent records a new event in the outbox.
	CreateTodoEvent(ctx context.Context, event TodoEvent) error
	// CreateChatEvent records a new chat message event in the outbox.
	CreateChatEvent(ctx context.Context, event ChatMessageEvent) error
	// FetchPendingEvents retrieves a batch of pending outbox events.
	FetchPendingEvents(ctx context.Context, limit int) ([]Event, error)
	// UpdateEvent updates the status, retry count, and last error of an outbox event.
	UpdateEvent(ctx context.Context, eventID uuid.UUID, status Status, retryCount int, lastError string) error
	// DeleteEvent deletes an event from the outbox.
	DeleteEvent(ctx context.Context, eventID uuid.UUID) error
}
