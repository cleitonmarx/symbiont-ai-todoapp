package postgres

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestAdvisoryLocker_TryLock(t *testing.T) {
	t.Parallel()

	lockName := "conversation-title:00000000-0000-0000-0000-000000000001"
	lockKey := advisoryLockKey(lockName)

	tests := map[string]struct {
		expect    func(sqlmock.Sqlmock)
		wantLock  bool
		wantErr   bool
		runUnlock bool
	}{
		"acquired": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT pg_try_advisory_lock($1)").
					WithArgs(lockKey).
					WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
				m.ExpectExec("SELECT pg_advisory_unlock($1)").
					WithArgs(lockKey).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantLock:  true,
			wantErr:   false,
			runUnlock: true,
		},
		"not-acquired": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT pg_try_advisory_lock($1)").
					WithArgs(lockKey).
					WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(false))
			},
			wantLock: false,
			wantErr:  false,
		},
		"query-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT pg_try_advisory_lock($1)").
					WithArgs(lockKey).
					WillReturnError(errors.New("db unavailable"))
			},
			wantLock: false,
			wantErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() //nolint:errcheck

			tt.expect(mock)

			locker := NewAdvisoryLocker(db)
			unlock, locked, err := locker.TryLock(t.Context(), lockName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.False(t, locked)
				assert.Nil(t, unlock)
				assert.NoError(t, mock.ExpectationsWereMet())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantLock, locked)

			if tt.runUnlock {
				assert.NotNil(t, unlock)
				unlock()
			} else {
				assert.Nil(t, unlock)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAdvisoryLockKey(t *testing.T) {
	t.Parallel()

	keyA := "conversation-title:abc"
	keyB := "conversation-title:def"

	assert.NotEqual(t, advisoryLockKey(keyA), advisoryLockKey(keyB))
	assert.Equal(t, advisoryLockKey(keyA), advisoryLockKey(keyA))
}
