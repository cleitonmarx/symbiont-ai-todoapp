package composite

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitCompositeActionRegistry is the initializer for CompositeActionRegistry, composing local and MCP gateway registries.
type InitCompositeActionRegistry struct {
	Local assistant.ActionRegistry `resolve:"local"`
	MCP   assistant.ActionRegistry `resolve:"mcp"`
}

// Initialize creates a CompositeActionRegistry from the local and MCP gateway registries and registers it in the dependency container.
func (i InitCompositeActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	composite := NewCompositeActionRegistry(ctx, i.Local, i.MCP)
	depend.Register[assistant.ActionRegistry](composite)
	return ctx, nil
}
