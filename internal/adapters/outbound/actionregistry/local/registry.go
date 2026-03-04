package local

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry/local/actions"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/cleitonmarx/symbiont/depend"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LocalRegistry manages a set of assistant actions defined within the todo application.
type LocalRegistry struct {
	actionsByName map[string]domain.AssistantAction
}

// NewActionRegistry creates a local assistant action registry.
func NewActionRegistry(se domain.SemanticEncoder, embeddingModel string, actionVectorList ...domain.AssistantAction) LocalRegistry {
	actionsByName := make(map[string]domain.AssistantAction)
	for _, actionVector := range actionVectorList {
		actionsByName[actionVector.Definition().Name] = actionVector
	}

	return LocalRegistry{
		actionsByName: actionsByName,
	}
}

// Execute invokes the appropriate action.
func (r LocalRegistry) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	spanCtx, span := telemetry.Start(ctx, trace.WithAttributes(
		attribute.String("assistant_action", call.Name),
	))
	defer span.End()
	details, exists := r.actionsByName[call.Name]
	if !exists {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"unknown_action","details":"Action '%s' is not registered."}`, call.Name),
		}
	}

	return details.Execute(spanCtx, call, conversationHistory)
}

// GetDefinition returns one action definition by name.
func (r LocalRegistry) GetDefinition(actionName string) (domain.AssistantActionDefinition, bool) {
	details, exists := r.actionsByName[actionName]
	if !exists {
		return domain.AssistantActionDefinition{}, false
	}
	return details.Definition(), true
}

// GetRenderer returns one deterministic action result renderer by action name.
func (r LocalRegistry) GetRenderer(actionName string) (domain.ActionResultRenderer, bool) {
	details, exists := r.actionsByName[actionName]
	if !exists {
		return nil, false
	}
	return details.Renderer()
}

// StatusMessage returns a status message about the action execution.
func (r LocalRegistry) StatusMessage(actionName string) string {
	if action, ok := r.actionsByName[actionName]; ok {
		if msg := action.StatusMessage(); msg != "" {
			return msg
		}
	}
	return "⏳ Processing request..."
}

type InitLocalActionRegistry struct {
	Uow             domain.UnitOfWork          `resolve:""`
	TodoCreator     usecases.TodoCreator       `resolve:""`
	TodoUpdater     usecases.TodoUpdater       `resolve:""`
	TodoDeleter     usecases.TodoDeleter       `resolve:""`
	TodoRepo        domain.TodoRepository      `resolve:""`
	SemanticEncoder domain.SemanticEncoder     `resolve:""`
	TimeProvider    domain.CurrentTimeProvider `resolve:""`
	EmbeddingModel  string                     `config:"LLM_EMBEDDING_MODEL"`
}

func (i InitLocalActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	actions := []domain.AssistantAction{
		actions.NewUIFiltersSetterAction(),
		actions.NewTodoFetcherAction(
			i.TodoRepo,
			i.SemanticEncoder,
			i.EmbeddingModel,
		),
		actions.NewBulkTodoCreatorAction(
			i.Uow,
			i.TodoCreator,
			i.TimeProvider,
		),
		actions.NewBulkTodoUpdaterAction(
			i.Uow,
			i.TodoUpdater,
		),
		actions.NewBulkTodoDueDateUpdaterAction(
			i.Uow,
			i.TodoUpdater,
			i.TimeProvider,
		),
		actions.NewBulkTodoDeleterAction(
			i.Uow,
			i.TodoDeleter,
		),
	}

	actionRegistry := NewActionRegistry(i.SemanticEncoder, i.EmbeddingModel, actions...)
	depend.RegisterNamed[domain.AssistantActionRegistry](actionRegistry, "local")
	return ctx, nil
}
