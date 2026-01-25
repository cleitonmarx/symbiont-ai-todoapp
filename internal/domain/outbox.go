package domain

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// OutboxEvent represents an event stored in the outbox.
type OutboxEvent struct {
	ID         uuid.UUID
	EntityType string
	EntityID   uuid.UUID
	Topic      string
	EventType  string
	Payload    []byte
	RetryCount int
	MaxRetries int
	LastError  sql.NullString
	CreatedAt  time.Time
}

// OutboxRepository defines the interface for managing outbox events.
type OutboxRepository interface {
	// RecordEvent records a new event in the outbox.
	RecordEvent(ctx context.Context, event TodoEvent) error
	// FetchPendingEvents retrieves a batch of pending outbox events.
	FetchPendingEvents(ctx context.Context, limit int) ([]OutboxEvent, error)
	// UpdateEvent updates the status, retry count, and last error of an outbox event.
	UpdateEvent(ctx context.Context, eventID uuid.UUID, status string, retryCount int, lastError string) error
	// DeleteEvent deletes an event from the outbox.
	DeleteEvent(ctx context.Context, eventID uuid.UUID) error
}
