package usecases

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
)

type UpdateTodo interface {
	Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.TodoStatus, dueDate *time.Time) (domain.Todo, error)
}

// UpdateTodoImpl is the implementation of the UpdateTodo use case.
type UpdateTodoImpl struct {
	uow          domain.UnitOfWork          `resolve:""`
	timeProvider domain.CurrentTimeProvider `resolve:""`
}

// NewUpdateTodoImpl creates a new instance of UpdateTodoImpl.
func NewUpdateTodoImpl(uow domain.UnitOfWork, timeProvider domain.CurrentTimeProvider) UpdateTodoImpl {
	return UpdateTodoImpl{
		uow:          uow,
		timeProvider: timeProvider,
	}
}

// Execute updates an existing todo item identified by id with the provided title and/or status.
func (uti UpdateTodoImpl) Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.TodoStatus, dueDate *time.Time) (domain.Todo, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	now := uti.timeProvider.Now()
	if err := validateUpdateTodoInputParams(title, status, dueDate, now); tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}
	var todo domain.Todo
	err := uti.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		td, err := uow.Todo().GetTodo(spanCtx, id)
		if err != nil {
			return err
		}

		if title != nil {
			td.Title = *title
		}

		if status != nil {
			td.Status = *status
		}
		if dueDate != nil {
			td.DueDate = *dueDate
		}

		td.UpdatedAt = now

		err = uow.Todo().UpdateTodo(spanCtx, td)
		if err != nil {
			return err
		}

		todo = td

		return uow.Outbox().RecordEvent(spanCtx, domain.TodoEvent{
			Type:   domain.TodoEventType_TODO_UPDATED,
			TodoID: todo.ID,
		})
	})

	if tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	return todo, nil
}

func validateUpdateTodoInputParams(title *string, status *domain.TodoStatus, dueDate *time.Time, now time.Time) error {
	if title != nil {
		if len(*title) < 3 || len(*title) > 200 {
			err := domain.NewValidationErr("title must be between 3 and 200 characters")
			return err
		}
	}

	if dueDate != nil {
		if dueDate.Truncate(24 * time.Hour).Before(now.Add(-48 * time.Hour).Truncate(24 * time.Hour)) {
			err := domain.NewValidationErr("due_date cannot be in the past 2 days")
			return err
		}
	}

	if status != nil {
		if *status != domain.TodoStatus_OPEN && *status != domain.TodoStatus_DONE {
			err := domain.NewValidationErr("invalid status value")
			return err
		}
	}

	return nil
}

// InitUpdateTodo initializes the UpdateTodo use case and registers it in the dependency container.
type InitUpdateTodo struct {
	Uow         domain.UnitOfWork          `resolve:""`
	TimeService domain.CurrentTimeProvider `resolve:""`
}

// Initialize initializes the UpdateTodoImpl use case.
func (iut InitUpdateTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[UpdateTodo](NewUpdateTodoImpl(iut.Uow, iut.TimeService))
	return ctx, nil
}
