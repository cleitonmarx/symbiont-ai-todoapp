package usecases

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
)

// CreateTodo defines the interface for the CreateTodo use case.
type CreateTodo interface {
	Execute(ctx context.Context, title string, dueDate time.Time) (domain.Todo, error)
}

// CreateTodoImpl is the implementation of the CreateTodo use case.
type CreateTodoImpl struct {
	uow          domain.UnitOfWork
	timeProvider domain.CurrentTimeProvider
	createUUID   func() uuid.UUID
}

// NewCreateTodoImpl creates a new instance of CreateTodoImpl.
func NewCreateTodoImpl(uow domain.UnitOfWork, timeProvider domain.CurrentTimeProvider) CreateTodoImpl {
	return CreateTodoImpl{
		uow:          uow,
		timeProvider: timeProvider,
		createUUID:   uuid.New,
	}
}

// Execute creates a new todo item.
func (cti CreateTodoImpl) Execute(ctx context.Context, title string, dueDate time.Time) (domain.Todo, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	now := cti.timeProvider.Now()
	if err := validateCreateTodoInputParams(title, dueDate, now); tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	todo := domain.Todo{
		ID:        cti.createUUID(),
		Title:     title,
		Status:    domain.TodoStatus_OPEN,
		DueDate:   dueDate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := cti.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		err := uow.Todo().CreateTodo(spanCtx, todo)
		if err != nil {
			return err
		}

		err = uow.Outbox().RecordEvent(spanCtx, domain.TodoEvent{
			Type:      domain.TodoEventType_TODO_CREATED,
			TodoID:    todo.ID,
			CreatedAt: now,
		})
		return err
	}); tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	return todo, nil
}

func validateCreateTodoInputParams(title string, dueDate time.Time, today time.Time) error {
	if len(title) < 3 || len(title) > 200 {
		err := domain.NewValidationErr("title must be between 3 and 200 characters")
		return err
	}

	if dueDate.Truncate(24 * time.Hour).Before(today.Add(-48 * time.Hour).Truncate(24 * time.Hour)) {
		err := domain.NewValidationErr("due_date cannot be in the past 2 days")
		return err
	}

	return nil
}

// InitCreateTodo initializes the CreateTodo use case and registers it in the dependency container.
type InitCreateTodo struct {
	Uow         domain.UnitOfWork          `resolve:""`
	TimeService domain.CurrentTimeProvider `resolve:""`
}

// Initialize initializes the CreateTodoImpl use case and registers it in the dependency container.
func (ict InitCreateTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[CreateTodo](NewCreateTodoImpl(ict.Uow, ict.TimeService))
	return ctx, nil
}
