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
	llmClient    domain.LLMClient
	llmModel     string
}

// NewCreateTodoImpl creates a new instance of CreateTodoImpl.
func NewCreateTodoImpl(uow domain.UnitOfWork, timeProvider domain.CurrentTimeProvider, llmClient domain.LLMClient, llmModel string) CreateTodoImpl {
	return CreateTodoImpl{
		uow:          uow,
		timeProvider: timeProvider,
		createUUID:   uuid.New,
		llmClient:    llmClient,
		llmModel:     llmModel,
	}
}

// Execute creates a new todo item.
func (cti CreateTodoImpl) Execute(ctx context.Context, title string, dueDate time.Time) (domain.Todo, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	now := cti.timeProvider.Now()

	todo := domain.Todo{
		ID:        cti.createUUID(),
		Title:     title,
		Status:    domain.TodoStatus_OPEN,
		DueDate:   dueDate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := todo.Validate(now); tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	embedding, err := cti.llmClient.Embed(spanCtx, cti.llmModel, todo.ToLLMInput())
	if err != nil {
		return domain.Todo{}, err
	}
	todo.Embedding = embedding

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

// InitCreateTodo initializes the CreateTodo use case and registers it in the dependency container.
type InitCreateTodo struct {
	Uow         domain.UnitOfWork          `resolve:""`
	TimeService domain.CurrentTimeProvider `resolve:""`
	LLMClient   domain.LLMClient           `resolve:""`
	Model       string                     `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize initializes the CreateTodoImpl use case and registers it in the dependency container.
func (ict InitCreateTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[CreateTodo](NewCreateTodoImpl(ict.Uow, ict.TimeService, ict.LLMClient, ict.Model))
	return ctx, nil
}
