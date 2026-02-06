package usecases

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
)

// TodoDeleter defines the interface for deleting todos within a unit of work.
type TodoDeleter interface {
	Delete(ctx context.Context, uow domain.UnitOfWork, id uuid.UUID) error
}

// TodoDeleterImpl is the implementation of the TodoDeleter interface.
type TodoDeleterImpl struct {
	uow          domain.UnitOfWork
	timeProvider domain.CurrentTimeProvider
}

// NewTodoDeleterImpl creates a new instance of TodoDeleterImpl.
func NewTodoDeleterImpl(uow domain.UnitOfWork, timeProvider domain.CurrentTimeProvider) TodoDeleterImpl {
	return TodoDeleterImpl{
		uow:          uow,
		timeProvider: timeProvider,
	}
}

// Delete deletes a todo item by its ID.
func (dt TodoDeleterImpl) Delete(ctx context.Context, uow domain.UnitOfWork, id uuid.UUID) error {
	_, found, err := uow.Todo().GetTodo(ctx, id) // Ensure the todo exists
	if err != nil {
		return err
	}
	if !found {
		return domain.NewNotFoundErr(fmt.Sprintf("todo with ID %s not found", id))
	}
	err = uow.Todo().DeleteTodo(ctx, id)
	if err != nil {
		return err
	}

	return uow.Outbox().CreateEvent(ctx, domain.TodoEvent{
		Type:      domain.TodoEventType_TODO_DELETED,
		TodoID:    id,
		CreatedAt: dt.timeProvider.Now(),
	})
}

// InitTodoDeleter initializes the TodoDeleter.
type InitTodoDeleter struct {
	Uow          domain.UnitOfWork          `resolve:""`
	TimeProvider domain.CurrentTimeProvider `resolve:""`
}

// Initialize registers the TodoDeleter use case in the dependency container.
func (i InitTodoDeleter) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[TodoDeleter](NewTodoDeleterImpl(
		i.Uow,
		i.TimeProvider,
	))
	return ctx, nil
}
