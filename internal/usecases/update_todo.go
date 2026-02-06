package usecases

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// UpdateTodo defines the interface for the UpdateTodo use case.
type UpdateTodo interface {
	Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.TodoStatus, dueDate *time.Time) (domain.Todo, error)
}

// UpdateTodoImpl is the implementation of the UpdateTodo use case.
type UpdateTodoImpl struct {
	uow      domain.UnitOfWork
	modifier TodoUpdater
}

// NewUpdateTodoImpl creates a new instance of UpdateTodoImpl.
func NewUpdateTodoImpl(uow domain.UnitOfWork, modifier TodoUpdater) UpdateTodoImpl {
	return UpdateTodoImpl{
		uow:      uow,
		modifier: modifier,
	}
}

// Execute updates an existing todo item identified by id with the provided title and/or status.
func (uti UpdateTodoImpl) Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.TodoStatus, dueDate *time.Time) (domain.Todo, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	var todo domain.Todo
	err := uti.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		td, err := uti.modifier.Update(spanCtx, uow, id, title, status, dueDate)
		if err != nil {
			return err
		}
		todo = td
		return nil
	})

	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	return todo, nil
}

// InitUpdateTodo initializes the UpdateTodo use case and registers it in the dependency container.
type InitUpdateTodo struct {
	Uow          domain.UnitOfWork `resolve:""`
	TodoModifier TodoUpdater       `resolve:""`
}

// Initialize initializes the UpdateTodoImpl use case.
func (iut InitUpdateTodo) Initialize(ctx context.Context) (context.Context, error) {
	uc := NewUpdateTodoImpl(iut.Uow, iut.TodoModifier)
	depend.Register[UpdateTodo](uc)

	return ctx, nil
}
