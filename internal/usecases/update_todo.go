package usecases

import (
	"context"
	"fmt"
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
	uow          domain.UnitOfWork
	timeProvider domain.CurrentTimeProvider
	llmClient    domain.LLMClient
	model        string
}

// NewUpdateTodoImpl creates a new instance of UpdateTodoImpl.
func NewUpdateTodoImpl(uow domain.UnitOfWork, timeProvider domain.CurrentTimeProvider, llmClient domain.LLMClient, model string) UpdateTodoImpl {
	return UpdateTodoImpl{
		uow:          uow,
		timeProvider: timeProvider,
		llmClient:    llmClient,
		model:        model,
	}
}

// Execute updates an existing todo item identified by id with the provided title and/or status.
func (uti UpdateTodoImpl) Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.TodoStatus, dueDate *time.Time) (domain.Todo, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	now := uti.timeProvider.Now()
	var todo domain.Todo
	err := uti.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		td, found, err := uow.Todo().GetTodo(spanCtx, id)
		if err != nil {
			return err
		}
		if !found {
			return domain.NewNotFoundErr(fmt.Sprintf("todo with ID %s not found", id))
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

		if err := td.Validate(now); err != nil {
			return err
		}

		embedding, err := uti.llmClient.Embed(spanCtx, uti.model, td.ToLLMInput())
		if err != nil {
			return err
		}
		td.Embedding = embedding

		if err := uow.Todo().UpdateTodo(spanCtx, td); err != nil {
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

// InitUpdateTodo initializes the UpdateTodo use case and registers it in the dependency container.
type InitUpdateTodo struct {
	Uow         domain.UnitOfWork          `resolve:""`
	TimeService domain.CurrentTimeProvider `resolve:""`
	LLMClient   domain.LLMClient           `resolve:""`
	Model       string                     `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize initializes the UpdateTodoImpl use case.
func (iut InitUpdateTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[UpdateTodo](NewUpdateTodoImpl(iut.Uow, iut.TimeService, iut.LLMClient, iut.Model))
	return ctx, nil
}
