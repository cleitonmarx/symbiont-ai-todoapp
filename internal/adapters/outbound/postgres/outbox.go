package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
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

// OutboxRepository implements the domain.OutboxRepository interface for Postgres.
type OutboxRepository struct {
	sb squirrel.StatementBuilderType
}

// NewOutboxRepository creates a new instance of OutboxRepository.
func NewOutboxRepository(br squirrel.BaseRunner) OutboxRepository {
	return OutboxRepository{
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(br),
	}
}

// CreateTodoEvent records a new event in the outbox.
func (op OutboxRepository) CreateTodoEvent(ctx context.Context, event domain.TodoEvent) error {
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
			string(domain.OutboxEntityType_Todo),
			event.TodoID,
			string(domain.OutboxTopic_Todo),
			string(event.Type),
			contentJSON,
			string(domain.OutboxStatus_Pending),
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
func (op OutboxRepository) CreateChatEvent(ctx context.Context, event domain.ChatMessageEvent) error {
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
			string(domain.OutboxEntityType_ChatMessage),
			event.ChatMessageID,
			string(domain.OutboxTopic_ChatMessages),
			string(event.Type),
			contentJSON,
			string(domain.OutboxStatus_Pending),
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
func (op OutboxRepository) FetchPendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	rows, err := op.sb.
		Select(
			outboxEventFields...,
		).
		From("outbox_events").
		Where(squirrel.Eq{"status": string(domain.OutboxStatus_Pending)}).
		Where(squirrel.LtOrEq{"available_at": time.Now().UTC()}).
		OrderBy("available_at ASC", "created_at ASC").
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
		var (
			entityType  string
			topic       string
			eventType   string
			status      string
			lastError   sql.NullString
			dedupeKey   sql.NullString
			processedAt sql.NullTime
		)
		err := rows.Scan(
			&oe.ID,
			&entityType,
			&oe.EntityID,
			&topic,
			&eventType,
			&payloadBytes,
			&status,
			&oe.RetryCount,
			&oe.MaxRetries,
			&lastError,
			&dedupeKey,
			&oe.AvailableAt,
			&processedAt,
			&oe.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		oe.EntityType = domain.OutboxEntityType(entityType)
		oe.Topic = domain.OutboxTopic(topic)
		oe.EventType = domain.EventType(eventType)
		oe.Status = domain.OutboxStatus(status)
		oe.Payload = payloadBytes
		if lastError.Valid {
			oe.LastError = &lastError.String
		}
		if dedupeKey.Valid {
			oe.DedupeKey = &dedupeKey.String
		}
		if processedAt.Valid {
			processedAtValue := processedAt.Time
			oe.ProcessedAt = &processedAtValue
		}

		events = append(events, oe)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// UpdateEvent updates the status, retry count, and last error of an outbox event.
func (op OutboxRepository) UpdateEvent(ctx context.Context, eventID uuid.UUID, status domain.OutboxStatus, retryCount int, lastError string) error {
	qry := op.sb.Update("outbox_events").
		Set("status", string(status)).
		Set("retry_count", retryCount).
		Set("last_error", lastError).
		Where(squirrel.Eq{"id": eventID})

	switch status {
	case domain.OutboxStatus_Pending:
		qry = qry.
			Set("available_at", time.Now().UTC().Add(backoffDelay(retryCount))).
			Set("processed_at", nil)
	case domain.OutboxStatus_Processed:
		qry = qry.Set("processed_at", time.Now().UTC())
	}

	_, err := qry.ExecContext(ctx)

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
