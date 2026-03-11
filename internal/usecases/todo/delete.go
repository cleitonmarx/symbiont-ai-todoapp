package todo

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// Delete defines the interface for the delete use case.
type Delete interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// DeleteImpl is the implementation of the delete use case.
type DeleteImpl struct {
	uow     transaction.UnitOfWork
	deleter Deleter
}

// NewDelete creates a new instance of DeleteImpl.
func NewDelete(uow transaction.UnitOfWork, deleter Deleter) DeleteImpl {
	return DeleteImpl{
		uow:     uow,
		deleter: deleter,
	}
}

// Execute deletes a todo item by its ID.
func (dti DeleteImpl) Execute(ctx context.Context, id uuid.UUID) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	return dti.uow.Execute(spanCtx, func(uowCtx context.Context, scope transaction.Scope) error {
		return dti.deleter.Delete(uowCtx, scope, id)
	})
}
