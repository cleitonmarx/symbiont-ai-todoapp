package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
)

// DeleteTodo defines the interface for the DeleteTodo use case.
type DeleteTodo interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// DeleteTodoImpl is the implementation of the DeleteTodo use case.
type DeleteTodoImpl struct {
	uow     domain.UnitOfWork
	deleter TodoDeleter
}

// NewDeleteTodo creates a new instance of DeleteTodoImpl.
func NewDeleteTodo(uow domain.UnitOfWork, deleter TodoDeleter) DeleteTodoImpl {
	return DeleteTodoImpl{
		uow:     uow,
		deleter: deleter,
	}
}

// Execute deletes a todo item by its ID.
func (dti DeleteTodoImpl) Execute(ctx context.Context, id uuid.UUID) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	return dti.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		return dti.deleter.Delete(spanCtx, uow, id)
	})
}

// InitDeleteTodo initializes the DeleteTodo use case.
type InitDeleteTodo struct {
	Uow         domain.UnitOfWork `resolve:""`
	TodoDeleter TodoDeleter       `resolve:""`
}

// Initialize registers the DeleteTodo use case in the dependency container.
func (i InitDeleteTodo) Initialize(ctx context.Context) (context.Context, error) {
	uc := NewDeleteTodo(i.Uow, i.TodoDeleter)
	depend.Register[DeleteTodo](uc)
	return ctx, nil
}
