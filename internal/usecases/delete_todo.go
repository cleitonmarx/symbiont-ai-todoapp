package usecases

import (
	"context"
	"fmt"

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
	uow          domain.UnitOfWork
	timeProvider domain.CurrentTimeProvider
}

// NewDeleteTodo creates a new instance of DeleteTodoImpl.
func NewDeleteTodo(uow domain.UnitOfWork, timeProvider domain.CurrentTimeProvider) DeleteTodoImpl {
	return DeleteTodoImpl{
		uow:          uow,
		timeProvider: timeProvider,
	}
}

// Execute deletes a todo item by its ID.
func (dti DeleteTodoImpl) Execute(ctx context.Context, id uuid.UUID) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	return dti.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		_, found, err := uow.Todo().GetTodo(spanCtx, id) // Ensure the todo exists
		if err != nil {
			return err
		}
		if !found {
			return domain.NewNotFoundErr(fmt.Sprintf("todo with ID %s not found", id))
		}
		err = uow.Todo().DeleteTodo(spanCtx, id)
		if err != nil {
			return err
		}

		return uow.Outbox().RecordEvent(spanCtx, domain.TodoEvent{
			Type:      domain.TodoEventType_TODO_DELETED,
			TodoID:    id,
			CreatedAt: dti.timeProvider.Now(),
		})

	})
}

// InitDeleteTodo initializes the DeleteTodo use case.
type InitDeleteTodo struct {
	Uow          domain.UnitOfWork          `resolve:""`
	TimeProvider domain.CurrentTimeProvider `resolve:""`
}

// Initialize registers the DeleteTodo use case in the dependency container.
func (i InitDeleteTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[DeleteTodo](NewDeleteTodo(i.Uow, i.TimeProvider))
	return ctx, nil
}
