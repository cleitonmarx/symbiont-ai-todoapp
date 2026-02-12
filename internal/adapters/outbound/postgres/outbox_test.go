package postgres

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOutboxRepository_CreateEvent(t *testing.T) {
	event := domain.TodoEvent{
		TodoID:    uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Type:      domain.EventType_TODO_CREATED,
		CreatedAt: time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC),
	}
	dedupeKey := fmt.Sprintf("todo:%s:%s:%d", event.Type, event.TodoID.String(), event.CreatedAt.UnixNano())

	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO outbox_events (id,entity_type,entity_id,topic,event_type,payload,status,retry_count,max_retries,last_error,dedupe_key,available_at,processed_at,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING").
					WithArgs(
						sqlmock.AnyArg(), // id
						string(domain.OutboxEntityType_Todo),
						event.TodoID,
						string(domain.OutboxTopic_Todo),
						string(event.Type),
						sqlmock.AnyArg(), // payload json
						string(domain.OutboxStatus_Pending),
						0,
						5,
						nil,
						dedupeKey,
						event.CreatedAt,
						nil,
						event.CreatedAt,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: false,
		},
		"db-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO outbox_events (id,entity_type,entity_id,topic,event_type,payload,status,retry_count,max_retries,last_error,dedupe_key,available_at,processed_at,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING").
					WithArgs(
						sqlmock.AnyArg(),
						string(domain.OutboxEntityType_Todo),
						event.TodoID,
						string(domain.OutboxTopic_Todo),
						string(event.Type),
						sqlmock.AnyArg(),
						string(domain.OutboxStatus_Pending),
						0,
						5,
						nil,
						dedupeKey,
						event.CreatedAt,
						nil,
						event.CreatedAt,
					).
					WillReturnError(errors.New("db error"))
			},
			err: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.expect(mock)

			repo := NewOutboxRepository(db)
			gotErr := repo.CreateTodoEvent(context.Background(), event)
			if tt.err {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOutboxRepository_CreateChatEvent(t *testing.T) {
	event := domain.ChatMessageEvent{
		Type:           domain.EventType_CHAT_MESSAGE_SENT,
		ChatRole:       domain.ChatRole_Assistant,
		ChatMessageID:  uuid.MustParse("223e4567-e89b-12d3-a456-426614174000"),
		ConversationID: "global",
		IsToolSuccess:  true,
		CreatedAt:      time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC),
	}

	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO outbox_events (id,entity_type,entity_id,topic,event_type,payload,status,retry_count,max_retries,last_error,dedupe_key,available_at,processed_at,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING").
					WithArgs(
						sqlmock.AnyArg(), // id
						string(domain.OutboxEntityType_ChatMessage),
						event.ChatMessageID,
						string(domain.OutboxTopic_ChatMessages),
						string(event.Type),
						sqlmock.AnyArg(), // payload
						string(domain.OutboxStatus_Pending),
						0,
						5,
						nil,
						"chat:CHAT_MESSAGE.SENT:223e4567-e89b-12d3-a456-426614174000",
						event.CreatedAt,
						nil,
						event.CreatedAt,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: false,
		},
		"db-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO outbox_events (id,entity_type,entity_id,topic,event_type,payload,status,retry_count,max_retries,last_error,dedupe_key,available_at,processed_at,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING").
					WithArgs(
						sqlmock.AnyArg(),
						string(domain.OutboxEntityType_ChatMessage),
						event.ChatMessageID,
						string(domain.OutboxTopic_ChatMessages),
						string(event.Type),
						sqlmock.AnyArg(),
						string(domain.OutboxStatus_Pending),
						0,
						5,
						nil,
						"chat:CHAT_MESSAGE.SENT:223e4567-e89b-12d3-a456-426614174000",
						event.CreatedAt,
						nil,
						event.CreatedAt,
					).
					WillReturnError(errors.New("db error"))
			},
			err: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.expect(mock)

			repo := NewOutboxRepository(db)
			gotErr := repo.CreateChatEvent(context.Background(), event)
			if tt.err {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOutboxRepository_FetchPendingEvents(t *testing.T) {
	id1 := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	t1 := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		limit   int
		expect  func(sqlmock.Sqlmock)
		wantLen int
		wantErr bool
	}{
		"success": {
			limit: 2,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(outboxEventFields).
					AddRow(
						id1,
						"Todo",
						id1,
						"Todo",
						"TODO_CREATED",
						[]byte(`{"id":"123"}`),
						string(domain.OutboxStatus_Pending),
						1,
						5,
						nil,
						"dedupe-key-1",
						t1,
						nil,
						t1,
					)
				m.ExpectQuery("SELECT id, entity_type, entity_id, topic, event_type, payload, status, retry_count, max_retries, last_error, dedupe_key, available_at, processed_at, created_at FROM outbox_events WHERE status = $1 AND available_at <= $2 ORDER BY available_at ASC, created_at ASC LIMIT 2 FOR UPDATE SKIP LOCKED").
					WithArgs(string(domain.OutboxStatus_Pending), sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		"db-error": {
			limit: 1,
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, entity_type, entity_id, topic, event_type, payload, status, retry_count, max_retries, last_error, dedupe_key, available_at, processed_at, created_at FROM outbox_events WHERE status = $1 AND available_at <= $2 ORDER BY available_at ASC, created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED").
					WithArgs(string(domain.OutboxStatus_Pending), sqlmock.AnyArg()).
					WillReturnError(errors.New("db error"))
			},
			wantLen: 0,
			wantErr: true,
		},
		"scan-error": {
			limit: 1,
			expect: func(m sqlmock.Sqlmock) {
				// invalid UUID to trigger scan error
				rows := sqlmock.NewRows(outboxEventFields).
					AddRow(
						"not-a-uuid",
						"Todo",
						id1,
						"Todo",
						"TODO_CREATED",
						[]byte(`{}`),
						string(domain.OutboxStatus_Pending),
						1,
						5,
						nil,
						nil,
						t1,
						nil,
						t1,
					)
				m.ExpectQuery("SELECT id, entity_type, entity_id, topic, event_type, payload, status, retry_count, max_retries, last_error, dedupe_key, available_at, processed_at, created_at FROM outbox_events WHERE status = $1 AND available_at <= $2 ORDER BY available_at ASC, created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED").
					WithArgs(string(domain.OutboxStatus_Pending), sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: true,
		},
		"no-rows": {
			limit: 1,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(outboxEventFields)
				m.ExpectQuery("SELECT id, entity_type, entity_id, topic, event_type, payload, status, retry_count, max_retries, last_error, dedupe_key, available_at, processed_at, created_at FROM outbox_events WHERE status = $1 AND available_at <= $2 ORDER BY available_at ASC, created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED").
					WithArgs(string(domain.OutboxStatus_Pending), sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.expect(mock)

			repo := NewOutboxRepository(db)
			got, err := repo.FetchPendingEvents(context.Background(), tt.limit)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
				if tt.wantLen > 0 {
					assert.NotEmpty(t, got[0].Payload)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOutboxRepository_UpdateEvent(t *testing.T) {
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE outbox_events SET status = $1, retry_count = $2, last_error = $3, processed_at = $4 WHERE id = $5").
					WithArgs(string(domain.OutboxStatus_Processed), 1, "ok", sqlmock.AnyArg(), id).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: false,
		},
		"pending-retry-updates-available-at": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE outbox_events SET status = $1, retry_count = $2, last_error = $3, available_at = $4, processed_at = $5 WHERE id = $6").
					WithArgs(string(domain.OutboxStatus_Pending), 2, "retry", sqlmock.AnyArg(), nil, id).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: false,
		},
		"db-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE outbox_events SET status = $1, retry_count = $2, last_error = $3, processed_at = $4 WHERE id = $5").
					WithArgs(string(domain.OutboxStatus_Processed), 1, "ok", sqlmock.AnyArg(), id).
					WillReturnError(errors.New("db error"))
			},
			err: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.expect(mock)

			repo := NewOutboxRepository(db)
			status := domain.OutboxStatus_Processed
			retryCount := 1
			lastError := "ok"
			if name == "pending-retry-updates-available-at" {
				status = domain.OutboxStatus_Pending
				retryCount = 2
				lastError = "retry"
			}
			gotErr := repo.UpdateEvent(context.Background(), id, status, retryCount, lastError)
			if tt.err {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOutboxRepository_DeleteEvent(t *testing.T) {
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM outbox_events WHERE id = $1").
					WithArgs(id).
					WillReturnResult(driver.RowsAffected(1))
			},
			err: false,
		},
		"db-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM outbox_events WHERE id = $1").
					WithArgs(id).
					WillReturnError(errors.New("db error"))
			},
			err: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.expect(mock)

			repo := NewOutboxRepository(db)
			gotErr := repo.DeleteEvent(context.Background(), id)
			if tt.err {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
