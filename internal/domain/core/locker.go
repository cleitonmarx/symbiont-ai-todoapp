package core

import "context"

// Locker coordinates concurrent work using arbitrary string keys.
type Locker interface {
	// TryLock attempts to acquire a lock for one key.
	// It returns an unlock callback when the lock is acquired.
	TryLock(ctx context.Context, key string) (unlock func(), locked bool, err error)
}
