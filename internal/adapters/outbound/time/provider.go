package time

import (
	"time"
)

// CurrentTimeProvider is an implementation of core.CurrentTimeProvider using the standard time package.
type CurrentTimeProvider struct{}

// Now returns the current time.
func (ts CurrentTimeProvider) Now() time.Time {
	return time.Now().UTC()
}
