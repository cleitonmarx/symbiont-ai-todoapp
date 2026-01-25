package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
)

var (
	outboxEventFields = []string{
		"id",
		"entity_type",
		"entity_id",
		"topic",
		"event_type",
		"payload",
		"retry_count",
		"max_retries",
		"last_error",
		"created_at",
	}
)

type OutboxRepository struct {
	sb squirrel.StatementBuilderType
}

func NewOutboxRepository(br squirrel.BaseRunner) OutboxRepository {
	return OutboxRepository{
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(br),
	}
}

func (op OutboxRepository) RecordEvent(ctx context.Context, event domain.TodoEvent) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	// Marshal the content to JSON
	contentJSON, err := json.Marshal(event)
	if tracing.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to marshal summary content: %w", err)
	}

	_, err = op.sb.Insert("outbox_events").
		Columns(
			outboxEventFields...,
		).
		Values(
			uuid.New(),
			"Todo",
			event.TodoID,
			"Todo",
			string(event.Type),
			contentJSON,
			0,
			5,
			nil,
			event.CreatedAt,
		).
		ExecContext(spanCtx)

	if tracing.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}

// FetchPendingEvents retrieves a batch of pending outbox events from the database.
func (op OutboxRepository) FetchPendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	rows, err := op.sb.
		Select(
			outboxEventFields...,
		).
		From("outbox_events").
		Where(squirrel.Eq{"status": "PENDING"}).
		OrderBy("created_at ASC").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED").
		QueryContext(ctx)

	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var events []domain.OutboxEvent
	for rows.Next() {
		var oe domain.OutboxEvent
		var payloadBytes []byte
		err := rows.Scan(
			&oe.ID,
			&oe.EntityType,
			&oe.EntityID,
			&oe.Topic,
			&oe.EventType,
			&payloadBytes,
			&oe.RetryCount,
			&oe.MaxRetries,
			&oe.LastError,
			&oe.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		events = append(events, oe)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// UpdateEvent updates the status, retry count, and last error of an outbox event.
func (op OutboxRepository) UpdateEvent(ctx context.Context, eventID uuid.UUID, status string, retryCount int, lastError string) error {
	_, err := op.sb.
		Update("outbox_events").
		Set("status", status).
		Set("retry_count", retryCount).
		Set("last_error", lastError).
		Where(squirrel.Eq{"id": eventID}).
		ExecContext(ctx)

	return err
}

// DeleteEvent deletes an outbox event from the database.
func (op OutboxRepository) DeleteEvent(ctx context.Context, eventID uuid.UUID) error {
	_, err := op.sb.
		Delete("outbox_events").
		Where(squirrel.Eq{"id": eventID}).
		ExecContext(ctx)

	return err
}
