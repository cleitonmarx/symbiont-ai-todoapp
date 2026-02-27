package composite

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
)

// CompositeActionRegistry implements domain.AssistantActionRegistry interface.
// It aggregates actions from multiple EmbeddingActionRegistry instances.
type CompositeActionRegistry struct {
	registriesActions []domain.AssistantActionRegistry
}

// NewCompositeActionRegistry creates a new CompositeActionRegistry from the given embedding registries.
func NewCompositeActionRegistry(ctx context.Context, registries ...domain.AssistantActionRegistry) CompositeActionRegistry {
	return CompositeActionRegistry{
		registriesActions: registries,
	}
}

// Execute iterates through the composed registries to execute the given action call, returning the first successful result.
func (r CompositeActionRegistry) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()
	for _, actionRegistry := range r.registriesActions {
		_, found := actionRegistry.GetDefinition(call.Name)
		if !found {
			continue
		}
		return actionRegistry.Execute(spanCtx, call, conversationHistory)
	}
	return domain.AssistantMessage{
		Role:    domain.ChatRole_Tool,
		Content: fmt.Sprintf("error: no registry found for action '%s'", call.Name),
	}
}

// GetDefinition returns one action definition by name.
func (r CompositeActionRegistry) GetDefinition(actionName string) (domain.AssistantActionDefinition, bool) {
	for _, actionRegistry := range r.registriesActions {
		definition, found := actionRegistry.GetDefinition(actionName)
		if found {
			return definition, true
		}
	}
	return domain.AssistantActionDefinition{}, false
}

// StatusMessage iterates through the composed registries to get the status message for the given action, returning a default message if none found.
func (r CompositeActionRegistry) StatusMessage(actionName string) string {
	for _, actionRegistry := range r.registriesActions {
		_, found := actionRegistry.GetDefinition(actionName)
		if found {
			return actionRegistry.StatusMessage(actionName)
		}
	}
	return "⏳ Processing request..."
}

// InitCompositeActionRegistry is the initializer for CompositeActionRegistry, composing local and MCP gateway registries.
type InitCompositeActionRegistry struct {
	Local domain.AssistantActionRegistry `resolve:"local"`
	MCP   domain.AssistantActionRegistry `resolve:"mcp"`
}

// Initialize creates a CompositeActionRegistry from the local and MCP gateway registries and registers it in the dependency container.
func (i InitCompositeActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	composite := NewCompositeActionRegistry(ctx, i.Local, i.MCP)
	depend.Register[domain.AssistantActionRegistry](composite)
	return ctx, nil
}
