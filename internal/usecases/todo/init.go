package todo

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitCreateTodo initializes the Create use case and registers it in the dependency container.
type InitCreateTodo struct {
	Uow     transaction.UnitOfWork `resolve:""`
	Creator Creator                `resolve:""`
}

// InitDeleteTodo initializes the Delete use case.
type InitDeleteTodo struct {
	Uow     transaction.UnitOfWork `resolve:""`
	Deleter Deleter                `resolve:""`
}

// InitListTodos initializes the List use case and registers it in the dependency container.
type InitListTodos struct {
	TodoRepo       domain.Repository `resolve:""`
	Encoder        semantic.Encoder  `resolve:""`
	EmbeddingModel string            `config:"LLM_EMBEDDING_MODEL"`
}

// InitCreator initializes the Creator and registers it in the dependency container.
type InitCreator struct {
	TimeService core.CurrentTimeProvider `resolve:""`
	Encoder     semantic.Encoder         `resolve:""`
	Model       string                   `config:"LLM_EMBEDDING_MODEL"`
}

// InitDeleter initializes the Deleter.
type InitDeleter struct {
	TimeProvider core.CurrentTimeProvider `resolve:""`
}

// InitUpdater initializes the Updater and registers it in the dependency container.
type InitUpdater struct {
	TimeService core.CurrentTimeProvider `resolve:""`
	Encoder     semantic.Encoder         `resolve:""`
	Model       string                   `config:"LLM_EMBEDDING_MODEL"`
}

// InitUpdateTodo initializes the Update use case and registers it in the dependency container.
type InitUpdateTodo struct {
	Uow          transaction.UnitOfWork `resolve:""`
	TodoModifier Updater                `resolve:""`
}

// Initialize registers the Create use case in the dependency container.
func (ict InitCreateTodo) Initialize(ctx context.Context) (context.Context, error) {
	uc := NewCreateImpl(ict.Uow, ict.Creator)
	depend.Register[Create](uc)
	return ctx, nil
}

// Initialize registers the Delete use case in the dependency container.
func (i InitDeleteTodo) Initialize(ctx context.Context) (context.Context, error) {
	uc := NewDelete(i.Uow, i.Deleter)
	depend.Register[Delete](uc)
	return ctx, nil
}

// Initialize registers the List use case in the dependency container.
func (ilt InitListTodos) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[List](NewListImpl(ilt.TodoRepo, ilt.Encoder, ilt.EmbeddingModel))
	return ctx, nil
}

// Initialize registers the Creator in the dependency container.
func (ict InitCreator) Initialize(ctx context.Context) (context.Context, error) {
	uc := NewCreatorImpl(ict.TimeService, ict.Encoder, ict.Model)
	depend.Register[Creator](uc)
	return ctx, nil
}

// Initialize registers the Deleter in the dependency container.
func (i InitDeleter) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[Deleter](NewDeleterImpl(i.TimeProvider))
	return ctx, nil
}

// Initialize registers the Updater in the dependency container.
func (itu InitUpdater) Initialize(ctx context.Context) (context.Context, error) {
	todoUpdater := NewUpdaterImpl(
		itu.TimeService,
		itu.Encoder,
		itu.Model,
	)
	depend.Register[Updater](todoUpdater)
	return ctx, nil
}

// Initialize registers the Update use case in the dependency container.
func (iut InitUpdateTodo) Initialize(ctx context.Context) (context.Context, error) {
	uc := NewUpdateImpl(iut.Uow, iut.TodoModifier)
	depend.Register[Update](uc)
	return ctx, nil
}
