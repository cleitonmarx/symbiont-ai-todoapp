package todo

import (
	"context"
	"time"

	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// Update defines the interface for the update use case.
type Update interface {
	Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.Status, dueDate *time.Time) (domain.Todo, error)
}

// UpdateImpl is the implementation of the update use case.
type UpdateImpl struct {
	uow      transaction.UnitOfWork
	modifier Updater
}

// NewUpdateImpl creates a new instance of UpdateImpl.
func NewUpdateImpl(uow transaction.UnitOfWork, modifier Updater) UpdateImpl {
	return UpdateImpl{
		uow:      uow,
		modifier: modifier,
	}
}

// Execute updates an existing todo item identified by id with the provided title, status, and/or due date.
func (uti UpdateImpl) Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.Status, dueDate *time.Time) (domain.Todo, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	var todo domain.Todo
	err := uti.uow.Execute(spanCtx, func(uowCtx context.Context, scope transaction.Scope) error {
		td, err := uti.modifier.Update(uowCtx, scope, id, title, status, dueDate)
		if err != nil {
			return err
		}
		todo = td
		return nil
	})

	if telemetry.IsErrorRecorded(span, err) {
		return domain.Todo{}, err
	}

	return todo, nil
}
