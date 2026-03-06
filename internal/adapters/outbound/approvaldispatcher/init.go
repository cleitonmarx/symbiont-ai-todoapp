package approvaldispatcher

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitDispatcher is used to initialize and register the dispatcher.
type InitDispatcher struct{}

// Initialize creates and registers the dispatcher in the dependency container.
func (i InitDispatcher) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[assistant.ActionApprovalDispatcher](NewDispatcher())
	return ctx, nil
}
