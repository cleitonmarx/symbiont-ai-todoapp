package postgres

import (
	"context"
	"database/sql"
	"hash/fnv"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const advisoryLockTimeout = 5 * time.Second

// AdvisoryLocker is a PostgreSQL advisory-lock implementation of core.Locker.
type AdvisoryLocker struct {
	db *sql.DB
}

// NewAdvisoryLocker creates a new AdvisoryLocker.
func NewAdvisoryLocker(db *sql.DB) AdvisoryLocker {
	return AdvisoryLocker{db: db}
}

// TryLock attempts to acquire a non-blocking advisory lock for one key.
func (l AdvisoryLocker) TryLock(ctx context.Context, key string) (func(), bool, error) {
	spanCtx, span := telemetry.StartSpan(ctx, trace.WithAttributes(
		attribute.String("lock.key", key),
	))
	defer span.End()

	conn, err := l.db.Conn(spanCtx)
	if err != nil {
		return nil, false, err
	}

	lockKey := advisoryLockKey(key)

	var locked bool
	err = conn.QueryRowContext(spanCtx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&locked)
	if err != nil {
		_ = conn.Close()
		return nil, false, err
	}

	if !locked {
		_ = conn.Close()
		return nil, false, nil
	}

	unlock := func() {
		if !locked {
			return
		}
		unlockCtx, unlockSpan := telemetry.StartSpan(ctx, trace.WithAttributes(
			attribute.String("unlock.key", key),
			attribute.Bool("lock.released", true),
		))
		defer unlockSpan.End()

		unlockCtx, cancel := context.WithTimeout(unlockCtx, advisoryLockTimeout)
		defer cancel()

		_, _ = conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", lockKey)
		_ = conn.Close()
	}

	return unlock, true, nil
}

// advisoryLockKey generates a consistent int64 key for a given string key using FNV hashing.
func advisoryLockKey(key string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return int64(h.Sum64())
}
