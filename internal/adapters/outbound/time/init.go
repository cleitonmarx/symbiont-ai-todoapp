package time

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitCurrentTimeProvider initializes the CurrentTimeProvider and registers it in the dependency container.
type InitCurrentTimeProvider struct{}

// Initialize registers the current time provider in the dependency container.
func (its InitCurrentTimeProvider) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[core.CurrentTimeProvider](CurrentTimeProvider{})
	return ctx, nil
}
