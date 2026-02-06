package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
)

// TodoUpdater defines the interface for modifying todo items.
type TodoUpdater interface {
	Update(ctx context.Context, uow domain.UnitOfWork, id uuid.UUID, title *string, status *domain.TodoStatus, dueDate *time.Time) (domain.Todo, error)
}

// TodoUpdaterImpl is the implementation of the TodoUpdater interface.
type TodoUpdaterImpl struct {
	uow          domain.UnitOfWork
	timeProvider domain.CurrentTimeProvider
	llmClient    domain.LLMClient
	model        string
}

// NewTodoUpdaterImpl creates a new instance of TodoUpdaterImpl.
func NewTodoUpdaterImpl(
	uow domain.UnitOfWork,
	timeProvider domain.CurrentTimeProvider,
	llmClient domain.LLMClient,
	model string,
) TodoUpdaterImpl {
	return TodoUpdaterImpl{
		uow:          uow,
		timeProvider: timeProvider,
		llmClient:    llmClient,
		model:        model,
	}
}

// Update modifies an existing todo item identified by id with the provided title and/or status.
func (tui TodoUpdaterImpl) Update(ctx context.Context, uow domain.UnitOfWork, id uuid.UUID, title *string, status *domain.TodoStatus, dueDate *time.Time) (domain.Todo, error) {
	now := tui.timeProvider.Now()
	var todo domain.Todo
	td, found, err := uow.Todo().GetTodo(ctx, id)
	if err != nil {
		return domain.Todo{}, err
	}
	if !found {
		return domain.Todo{}, domain.NewNotFoundErr(fmt.Sprintf("todo with ID %s not found", id))
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
		return domain.Todo{}, err
	}

	resp, err := tui.llmClient.Embed(ctx, tui.model, td.ToLLMInput())
	if err != nil {
		return domain.Todo{}, err
	}

	RecordLLMTokensEmbedding(ctx, resp.TotalTokens)
	td.Embedding = resp.Embedding

	if err := uow.Todo().UpdateTodo(ctx, td); err != nil {
		return domain.Todo{}, err
	}

	todo = td

	if err = uow.Outbox().CreateEvent(ctx, domain.TodoEvent{
		Type:   domain.TodoEventType_TODO_UPDATED,
		TodoID: todo.ID,
	}); err != nil {
		return domain.Todo{}, err
	}

	return todo, nil
}

// InitTodoUpdater initializes the TodoUpdater and registers it in the dependency container.
type InitTodoUpdater struct {
	Uow         domain.UnitOfWork          `resolve:""`
	TimeService domain.CurrentTimeProvider `resolve:""`
	LLMClient   domain.LLMClient           `resolve:""`
	Model       string                     `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize initializes the TodoUpdaterImpl use case.
func (itu InitTodoUpdater) Initialize(ctx context.Context) (context.Context, error) {
	todoUpdater := NewTodoUpdaterImpl(
		itu.Uow,
		itu.TimeService,
		itu.LLMClient,
		itu.Model,
	)
	depend.Register[TodoUpdater](todoUpdater)
	return ctx, nil
}
