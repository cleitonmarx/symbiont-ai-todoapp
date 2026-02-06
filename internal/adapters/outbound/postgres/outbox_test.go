package postgres

import (
	"context"
	"database/sql/driver"
	"errors"
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
		Type:      domain.TodoEventType("Created"),
		CreatedAt: time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC),
	}

	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO outbox_events (id,entity_type,entity_id,topic,event_type,payload,retry_count,max_retries,last_error,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)").
					WithArgs(
						sqlmock.AnyArg(), // id
						"Todo",
						event.TodoID,
						"Todo",
						string(event.Type),
						sqlmock.AnyArg(), // payload json
						0,
						5,
						nil,
						event.CreatedAt,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: false,
		},
		"db-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO outbox_events (id,entity_type,entity_id,topic,event_type,payload,retry_count,max_retries,last_error,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)").
					WithArgs(
						sqlmock.AnyArg(),
						"Todo",
						event.TodoID,
						"Todo",
						string(event.Type),
						sqlmock.AnyArg(),
						0,
						5,
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
			gotErr := repo.CreateEvent(context.Background(), event)
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
						"Created",
						[]byte(`{"id":"123"}`),
						1,
						5,
						nil,
						t1,
					)
				m.ExpectQuery("SELECT id, entity_type, entity_id, topic, event_type, payload, retry_count, max_retries, last_error, created_at FROM outbox_events WHERE status = $1 ORDER BY created_at ASC LIMIT 2 FOR UPDATE SKIP LOCKED").
					WithArgs("PENDING").
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		"db-error": {
			limit: 1,
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, entity_type, entity_id, topic, event_type, payload, retry_count, max_retries, last_error, created_at FROM outbox_events WHERE status = $1 ORDER BY created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED").
					WithArgs("PENDING").
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
						"Created",
						[]byte(`{}`),
						1,
						5,
						nil,
						t1,
					)
				m.ExpectQuery("SELECT id, entity_type, entity_id, topic, event_type, payload, retry_count, max_retries, last_error, created_at FROM outbox_events WHERE status = $1 ORDER BY created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED").
					WithArgs("PENDING").
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: true,
		},
		"no-rows": {
			limit: 1,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(outboxEventFields)
				m.ExpectQuery("SELECT id, entity_type, entity_id, topic, event_type, payload, retry_count, max_retries, last_error, created_at FROM outbox_events WHERE status = $1 ORDER BY created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED").
					WithArgs("PENDING").
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
				m.ExpectExec("UPDATE outbox_events SET status = $1, retry_count = $2, last_error = $3 WHERE id = $4").
					WithArgs("DONE", 1, "ok", id).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: false,
		},
		"db-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE outbox_events SET status = $1, retry_count = $2, last_error = $3 WHERE id = $4").
					WithArgs("DONE", 1, "ok", id).
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
			gotErr := repo.UpdateEvent(context.Background(), id, "DONE", 1, "ok")
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
