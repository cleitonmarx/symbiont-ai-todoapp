package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
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
		"status",
		"retry_count",
		"max_retries",
		"last_error",
		"dedupe_key",
		"available_at",
		"processed_at",
		"created_at",
	}
)

// Repository implements the event.Repository interface for Postgres.
type Repository struct {
	sb squirrel.StatementBuilderType
}

// NewOutboxRepository creates a new instance of Repository.
func NewOutboxRepository(br squirrel.BaseRunner) Repository {
	return Repository{
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(br),
	}
}

// CreateTodoEvent records a new event in the outbox.
func (op Repository) CreateTodoEvent(ctx context.Context, event outbox.TodoEvent) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	createdAt := event.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	// Marshal the content to JSON
	contentJSON, err := json.Marshal(event)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to marshal summary content: %w", err)
	}

	dedupeKey := fmt.Sprintf(
		"todo:%s:%s:%d",
		event.Type,
		event.TodoID.String(),
		createdAt.UnixNano(),
	)

	_, err = op.sb.Insert("outbox_events").
		Columns(
			outboxEventFields...,
		).
		Values(
			uuid.New(),
			string(outbox.EntityType_Todo),
			event.TodoID,
			string(outbox.Topic_Todo),
			string(event.Type),
			contentJSON,
			string(outbox.Status_Pending),
			0,
			5,
			nil,
			dedupeKey,
			createdAt,
			nil,
			createdAt,
		).
		Suffix("ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING").
		ExecContext(spanCtx)

	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}

// CreateChatEvent records a new chat message event in the outbox.
func (op Repository) CreateChatEvent(ctx context.Context, event outbox.ChatMessageEvent) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	createdAt := time.Now().UTC()

	contentJSON, err := json.Marshal(event)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to marshal chat event content: %w", err)
	}

	dedupeKey := fmt.Sprintf(
		"chat:%s:%s",
		event.Type,
		event.ChatMessageID.String(),
	)

	_, err = op.sb.Insert("outbox_events").
		Columns(
			outboxEventFields...,
		).
		Values(
			uuid.New(),
			string(outbox.EntityType_ChatMessage),
			event.ChatMessageID,
			string(outbox.Topic_ChatMessages),
			string(event.Type),
			contentJSON,
			string(outbox.Status_Pending),
			0,
			5,
			nil,
			dedupeKey,
			createdAt,
			nil,
			createdAt,
		).
		Suffix("ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING").
		ExecContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to insert chat outbox event: %w", err)
	}

	return nil
}

// FetchPendingEvents retrieves a batch of pending outbox events from the database.
func (op Repository) FetchPendingEvents(ctx context.Context, limit int) ([]outbox.Event, error) {
	rows, err := op.sb.
		Select(
			outboxEventFields...,
		).
		From("outbox_events").
		Where(squirrel.Eq{"status": string(outbox.Status_Pending)}).
		Where(squirrel.LtOrEq{"available_at": time.Now().UTC()}).
		OrderBy("available_at ASC", "created_at ASC").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED").
		QueryContext(ctx)

	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var events []outbox.Event
	for rows.Next() {
		var oe outbox.Event
		var payloadBytes []byte

		err := rows.Scan(
			&oe.ID,
			&oe.EntityType,
			&oe.EntityID,
			&oe.Topic,
			&oe.EventType,
			&payloadBytes,
			&oe.Status,
			&oe.RetryCount,
			&oe.MaxRetries,
			&oe.LastError,
			&oe.DedupeKey,
			&oe.AvailableAt,
			&oe.ProcessedAt,
			&oe.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		oe.Payload = payloadBytes

		events = append(events, oe)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// UpdateEvent updates the status, retry count, and last error of an outbox event.
func (op Repository) UpdateEvent(ctx context.Context, eventID uuid.UUID, status outbox.Status, retryCount int, lastError string) error {
	qry := op.sb.Update("outbox_events").
		Set("status", string(status)).
		Set("retry_count", retryCount).
		Set("last_error", lastError).
		Where(squirrel.Eq{"id": eventID})

	switch status {
	case outbox.Status_Pending:
		qry = qry.
			Set("available_at", time.Now().UTC().Add(backoffDelay(retryCount))).
			Set("processed_at", nil)
	case outbox.Status_Processed:
		qry = qry.Set("processed_at", time.Now().UTC())
	}

	_, err := qry.ExecContext(ctx)

	return err
}

// DeleteEvent deletes an outbox event from the database.
func (op Repository) DeleteEvent(ctx context.Context, eventID uuid.UUID) error {
	_, err := op.sb.
		Delete("outbox_events").
		Where(squirrel.Eq{"id": eventID}).
		ExecContext(ctx)

	return err
}

func backoffDelay(retryCount int) time.Duration {
	switch {
	case retryCount <= 1:
		return 2 * time.Second
	case retryCount == 2:
		return 5 * time.Second
	case retryCount == 3:
		return 15 * time.Second
	case retryCount == 4:
		return 30 * time.Second
	case retryCount == 5:
		return time.Minute
	default:
		return 2 * time.Minute
	}
}
