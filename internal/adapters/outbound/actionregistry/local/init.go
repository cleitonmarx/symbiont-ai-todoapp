package local

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry/local/actions"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitLocalActionRegistry initializes the local ActionRegistry with core and domain dependencies and registers it in the dependency container.
type InitLocalActionRegistry struct {
	Uow            transaction.UnitOfWork   `resolve:""`
	Creator        todouc.Creator           `resolve:""`
	Updater        todouc.Updater           `resolve:""`
	Deleter        todouc.Deleter           `resolve:""`
	TodoRepo       todo.Repository          `resolve:""`
	Encoder        semantic.Encoder         `resolve:""`
	TimeProvider   core.CurrentTimeProvider `resolve:""`
	EmbeddingModel string                   `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize creates a LocalActionRegistry with the provided dependencies and registers it in the dependency container.
func (i InitLocalActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	actions := []assistant.Action{
		actions.NewSetUIFiltersAction(),
		actions.NewFetchTodosAction(
			i.TodoRepo,
			i.Encoder,
			i.EmbeddingModel,
		),
		actions.NewCreateTodosAction(
			i.Uow,
			i.Creator,
			i.TimeProvider,
		),
		actions.NewUpdateTodosAction(
			i.Uow,
			i.Updater,
		),
		actions.NewUpdateTodosDueDateAction(
			i.Uow,
			i.Updater,
			i.TimeProvider,
		),
		actions.NewDeleteTodosAction(
			i.Uow,
			i.Deleter,
		),
	}

	actionRegistry := NewActionRegistry(i.Encoder, i.EmbeddingModel, actions...)
	depend.RegisterNamed[assistant.ActionRegistry](actionRegistry, "local")
	return ctx, nil
}
