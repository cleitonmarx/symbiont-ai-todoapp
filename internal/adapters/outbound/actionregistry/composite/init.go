package composite

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitActionRegistry is the initializer for ActionRegistry, composing local and MCP gateway registries.
type InitActionRegistry struct {
	Local assistant.ActionRegistry `resolve:"local"`
	MCP   assistant.ActionRegistry `resolve:"mcp"`
}

// Initialize creates an ActionRegistry from the local and MCP gateway registries and registers it in the dependency container.
func (i InitActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	composite := NewActionRegistry(ctx, i.Local, i.MCP)
	depend.Register[assistant.ActionRegistry](composite)
	return ctx, nil
}
