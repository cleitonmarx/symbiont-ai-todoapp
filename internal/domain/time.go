package domain

import "time"

// CurrentTimeProvider provides the current time.
type CurrentTimeProvider interface {
	Now() time.Time
}
