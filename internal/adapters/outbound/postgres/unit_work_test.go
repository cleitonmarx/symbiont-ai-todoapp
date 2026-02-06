package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUnitOfWork_Execute(t *testing.T) {
	todoID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := map[string]struct {
		setupMock func(sqlmock.Sqlmock)
		fn        func(uow domain.UnitOfWork) error
		expectErr bool
	}{
		"success-commit": {
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectExec("DELETE FROM todos WHERE id = $1").
					WithArgs(todoID).
					WillReturnResult(sqlmock.NewResult(0, 1))
				m.ExpectCommit()
			},
			fn: func(uow domain.UnitOfWork) error {
				return uow.Todo().DeleteTodo(context.Background(), todoID)
			},
			expectErr: false,
		},
		"success-rollback-on-error": {
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectExec("DELETE FROM todos WHERE id = $1").
					WithArgs(todoID).
					WillReturnError(errors.New("delete error"))
				m.ExpectRollback()
			},
			fn: func(uow domain.UnitOfWork) error {
				return uow.Todo().DeleteTodo(context.Background(), todoID)
			},
			expectErr: true,
		},
		"begin-transaction-error": {
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin().WillReturnError(errors.New("begin error"))
			},
			fn: func(uow domain.UnitOfWork) error {
				return nil
			},
			expectErr: true,
		},
		"commit-error": {
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectExec("DELETE FROM todos WHERE id = $1").
					WithArgs(todoID).
					WillReturnResult(sqlmock.NewResult(0, 1))
				m.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			fn: func(uow domain.UnitOfWork) error {
				return uow.Todo().DeleteTodo(context.Background(), todoID)
			},
			expectErr: true,
		},
		"rollback-error-with-original-error": {
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectExec("DELETE FROM todos WHERE id = $1").
					WithArgs(todoID).
					WillReturnError(errors.New("delete error"))
				m.ExpectRollback().WillReturnError(errors.New("rollback error"))
			},
			fn: func(uow domain.UnitOfWork) error {
				return uow.Todo().DeleteTodo(context.Background(), todoID)
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() //nolint:errcheck

			tt.setupMock(mock)

			uow := NewUnitOfWork(db)
			err = uow.Execute(context.Background(), tt.fn)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUnitOfWork_Todo(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close() //nolint:errcheck

	uow := NewUnitOfWork(db)
	repo := uow.Todo()

	assert.NotNil(t, repo)
	assert.IsType(t, TodoRepository{}, repo)
}

func TestUnitOfWork_Outbox(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close() //nolint:errcheck

	uow := NewUnitOfWork(db)
	outbox := uow.Outbox()

	assert.NotNil(t, outbox)
	assert.IsType(t, OutboxRepository{}, outbox)
}

func TestUnitOfWork_getBaseRunner(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close() //nolint:errcheck

	t.Run("returns-db-when-no-transaction", func(t *testing.T) {
		uow := NewUnitOfWork(db)
		runner := uow.getBaseRunner()
		assert.Equal(t, db, runner)
	})

	t.Run("returns-tx-when-in-transaction", func(t *testing.T) {
		mock.ExpectBegin()

		tx, err := db.Begin()
		assert.NoError(t, err)

		uow := &UnitOfWork{
			db: db,
			tx: tx,
		}

		runner := uow.getBaseRunner()
		assert.Equal(t, tx, runner)

		// Clean up
		mock.ExpectRollback()
		_ = tx.Rollback()
	})
}

func TestUnitOfWork_TransactionIsolation(t *testing.T) {
	todoID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	defer db.Close() //nolint:errcheck

	// Simulate nested operations within transaction
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM todos WHERE id = $1").
		WithArgs(todoID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO outbox_events (id,entity_type,entity_id,topic,event_type,payload,retry_count,max_retries,last_error,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)").
		WithArgs(
			sqlmock.AnyArg(),
			"Todo",
			todoID,
			"Todo",
			"Deleted",
			sqlmock.AnyArg(),
			0,
			5,
			nil,
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	uow := NewUnitOfWork(db)
	err = uow.Execute(context.Background(), func(uow domain.UnitOfWork) error {
		// Delete todo
		if err := uow.Todo().DeleteTodo(context.Background(), todoID); err != nil {
			return err
		}

		// Publish event - both should use same transaction
		event := domain.TodoEvent{
			TodoID:    todoID,
			Type:      domain.TodoEventType("Deleted"),
			CreatedAt: time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC),
		}
		return uow.Outbox().CreateEvent(context.Background(), event)
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInitUnitOfWork_Initialize(t *testing.T) {
	i := &InitUnitOfWork{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	_, err = depend.Resolve[domain.UnitOfWork]()
	assert.NoError(t, err)

}
