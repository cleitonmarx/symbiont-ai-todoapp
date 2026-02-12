package time

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
)

// CurrentTimeProvider is an implementation of domain.CurrentTimeProvider using the standard time package.
type CurrentTimeProvider struct{}

// Now returns the current time.
func (ts CurrentTimeProvider) Now() time.Time {
	return time.Now()
}

// InitCurrentTimeProvider initializes the CurrentTimeProvider and registers it in the dependency container.
type InitCurrentTimeProvider struct {
}

// Initialize registers the CurrentTimeProvider in the dependency container.
func (its InitCurrentTimeProvider) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.CurrentTimeProvider](CurrentTimeProvider{})
	return ctx, nil
}
