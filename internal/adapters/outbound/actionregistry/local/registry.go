package local

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ActionRegistry manages a set of assistant actions defined within the todo application.
type ActionRegistry struct {
	actionsByName map[string]assistant.Action
}

// NewActionRegistry creates a local assistant action registry.
func NewActionRegistry(se semantic.Encoder, embeddingModel string, actionVectorList ...assistant.Action) ActionRegistry {
	actionsByName := make(map[string]assistant.Action)
	for _, actionVector := range actionVectorList {
		actionsByName[actionVector.Definition().Name] = actionVector
	}

	return ActionRegistry{
		actionsByName: actionsByName,
	}
}

// Execute invokes the appropriate action.
func (r ActionRegistry) Execute(ctx context.Context, call assistant.ActionCall, conversationHistory []assistant.Message) assistant.Message {
	spanCtx, span := telemetry.StartSpan(ctx, trace.WithAttributes(
		attribute.String("assistant_action", call.Name),
	))
	defer span.End()
	details, exists := r.actionsByName[call.Name]
	if !exists {
		errMsg := fmt.Sprintf("Action '%s' is not registered.", call.Name)
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"unknown_action","details":"%s"}`, errMsg),
			ActionError:  &errMsg,
		}
	}
	return details.Execute(spanCtx, call, conversationHistory)
}

// GetDefinition returns one action definition by name.
func (r ActionRegistry) GetDefinition(actionName string) (assistant.ActionDefinition, bool) {
	details, exists := r.actionsByName[actionName]
	if !exists {
		return assistant.ActionDefinition{}, false
	}
	return details.Definition(), true
}

// GetRenderer returns one deterministic action result renderer by action name.
func (r ActionRegistry) GetRenderer(actionName string) (assistant.ActionResultRenderer, bool) {
	details, exists := r.actionsByName[actionName]
	if !exists {
		return nil, false
	}
	return details.Renderer()
}

// StatusMessage returns a status message about the action execution.
func (r ActionRegistry) StatusMessage(actionName string) string {
	if action, ok := r.actionsByName[actionName]; ok {
		if msg := action.StatusMessage(); msg != "" {
			return msg
		}
	}
	return "⏳ Processing request..."
}
