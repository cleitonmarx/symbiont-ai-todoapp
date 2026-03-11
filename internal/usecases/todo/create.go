package todo

import (
	"context"
	"time"

	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// Create defines the interface for the create use case.
type Create interface {
	Execute(ctx context.Context, title string, dueDate time.Time) (domain.Todo, error)
}

// CreateImpl is the implementation of the create use case.
type CreateImpl struct {
	uow     transaction.UnitOfWork
	creator Creator
}

// NewCreateImpl creates a new instance of CreateImpl.
func NewCreateImpl(uow transaction.UnitOfWork, creator Creator) CreateImpl {
	return CreateImpl{
		uow:     uow,
		creator: creator,
	}
}

// Execute creates a new todo item.
func (cti CreateImpl) Execute(ctx context.Context, title string, dueDate time.Time) (domain.Todo, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	var todo domain.Todo
	err := cti.uow.Execute(spanCtx, func(uowCtx context.Context, scope transaction.Scope) error {
		var err error
		todo, err = cti.creator.Create(uowCtx, scope, title, dueDate)
		return err
	})
	if telemetry.IsErrorRecorded(span, err) {
		return domain.Todo{}, err
	}

	return todo, err
}
