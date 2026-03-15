package composite

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// ActionRegistry implements assistant.ActionRegistry interface.
// It aggregates actions from multiple EmbeddingActionRegistry instances.
type ActionRegistry struct {
	registriesActions []assistant.ActionRegistry
}

// NewActionRegistry creates a new composite ActionRegistry from the given embedding registries.
func NewActionRegistry(ctx context.Context, registries ...assistant.ActionRegistry) ActionRegistry {
	return ActionRegistry{
		registriesActions: registries,
	}
}

// Execute iterates through the composed registries to execute the given action call, returning the first successful result.
func (r ActionRegistry) Execute(ctx context.Context, call assistant.ActionCall, conversationHistory []assistant.Message) assistant.Message {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()
	for _, actionRegistry := range r.registriesActions {
		_, found := actionRegistry.GetDefinition(call.Name)
		if !found {
			continue
		}
		return actionRegistry.Execute(spanCtx, call, conversationHistory)
	}
	errMsg := fmt.Sprintf("no registry found for action '%s'", call.Name)
	return assistant.Message{
		Role:        assistant.ChatRole_Tool,
		Content:     "error: " + errMsg,
		ActionError: &errMsg,
	}
}

// GetDefinition returns one action definition by name.
func (r ActionRegistry) GetDefinition(actionName string) (assistant.ActionDefinition, bool) {
	for _, actionRegistry := range r.registriesActions {
		definition, found := actionRegistry.GetDefinition(actionName)
		if found {
			return definition, true
		}
	}
	return assistant.ActionDefinition{}, false
}

// GetRenderer returns one deterministic action result renderer by action name.
func (r ActionRegistry) GetRenderer(actionName string) (assistant.ActionResultRenderer, bool) {
	for _, actionRegistry := range r.registriesActions {
		renderer, found := actionRegistry.GetRenderer(actionName)
		if found {
			return renderer, true
		}
	}
	return nil, false
}

// StatusMessage iterates through the composed registries to get the status message for the given action, returning a default message if none found.
func (r ActionRegistry) StatusMessage(actionName string) string {
	for _, actionRegistry := range r.registriesActions {
		_, found := actionRegistry.GetDefinition(actionName)
		if found {
			return actionRegistry.StatusMessage(actionName)
		}
	}
	return "⏳ Processing request..."
}
