package usecases

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
)

// TodoCreator defines the interface for creating todos within a unit of work.
type TodoCreator interface {
	Create(ctx context.Context, uow domain.UnitOfWork, title string, dueDate time.Time) (domain.Todo, error)
}

// TodoCreatorImpl is the implementation of the TodoCreator use case.
type TodoCreatorImpl struct {
	uow          domain.UnitOfWork
	timeProvider domain.CurrentTimeProvider
	createUUID   func() uuid.UUID
	llmClient    domain.LLMClient
	llmModel     string
}

// NewTodoCreatorImpl creates a new instance of TodoCreatorImpl.
func NewTodoCreatorImpl(uow domain.UnitOfWork, timeProvider domain.CurrentTimeProvider, llmClient domain.LLMClient, llmModel string) TodoCreatorImpl {
	return TodoCreatorImpl{
		uow:          uow,
		timeProvider: timeProvider,
		createUUID:   uuid.New,
		llmClient:    llmClient,
		llmModel:     llmModel,
	}
}

// Create creates a new todo item within the provided unit of work.
func (tci TodoCreatorImpl) Create(ctx context.Context, uow domain.UnitOfWork, title string, dueDate time.Time) (domain.Todo, error) {
	now := tci.timeProvider.Now()

	todo := domain.Todo{
		ID:        tci.createUUID(),
		Title:     title,
		Status:    domain.TodoStatus_OPEN,
		DueDate:   dueDate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := todo.Validate(now); err != nil {
		return domain.Todo{}, err
	}

	resp, err := tci.llmClient.Embed(ctx, tci.llmModel, todo.ToLLMInput())
	if err != nil {
		return domain.Todo{}, err
	}

	RecordLLMTokensEmbedding(ctx, resp.TotalTokens)
	todo.Embedding = resp.Embedding

	err = uow.Todo().CreateTodo(ctx, todo)
	if err != nil {
		return domain.Todo{}, err
	}

	err = uow.Outbox().CreateEvent(ctx, domain.TodoEvent{
		Type:      domain.TodoEventType_TODO_CREATED,
		TodoID:    todo.ID,
		CreatedAt: now,
	})
	if err != nil {
		return domain.Todo{}, err
	}

	return todo, nil
}

// InitTodoCreator initializes the TodoCreator and registers it in the dependency container.
type InitTodoCreator struct {
	Uow         domain.UnitOfWork          `resolve:""`
	TimeService domain.CurrentTimeProvider `resolve:""`
	LLMClient   domain.LLMClient           `resolve:""`
	Model       string                     `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize initializes the TodoCreatorImpl use case and registers it in the dependency container.
func (ict InitTodoCreator) Initialize(ctx context.Context) (context.Context, error) {
	uc := NewTodoCreatorImpl(ict.Uow, ict.TimeService, ict.LLMClient, ict.Model)
	depend.Register[TodoCreator](uc)
	return ctx, nil
}
