package usecases

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
)

// CreateTodo defines the interface for the CreateTodo use case.
type CreateTodo interface {
	Execute(ctx context.Context, title string, dueDate time.Time) (domain.Todo, error)
}

type CreateTodoImpl struct {
	uow     domain.UnitOfWork
	creator TodoCreator
}

// NewCreateTodoImpl creates a new instance of CreateTodoImpl.
func NewCreateTodoImpl(uow domain.UnitOfWork, creator TodoCreator) CreateTodoImpl {
	return CreateTodoImpl{
		uow:     uow,
		creator: creator,
	}
}

// Execute creates a new todo item.
func (cti CreateTodoImpl) Execute(ctx context.Context, title string, dueDate time.Time) (domain.Todo, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	var todo domain.Todo
	err := cti.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		var err error
		todo, err = cti.creator.Create(spanCtx, uow, title, dueDate)
		return err
	})
	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	return todo, err
}

// InitCreateTodo initializes the CreateTodo use case and registers it in the dependency container.
type InitCreateTodo struct {
	Uow     domain.UnitOfWork `resolve:""`
	Creator TodoCreator       `resolve:""`
}

// Initialize initializes the CreateTodoImpl use case and registers it in the dependency container.
func (ict InitCreateTodo) Initialize(ctx context.Context) (context.Context, error) {
	uc := NewCreateTodoImpl(ict.Uow, ict.Creator)
	depend.Register[CreateTodo](uc)
	return ctx, nil
}
