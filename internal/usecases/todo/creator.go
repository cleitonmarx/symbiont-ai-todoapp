package todo

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/metrics"
	"github.com/google/uuid"
)

// Creator defines the interface for creating todos within a unit of work scope.
type Creator interface {
	Create(ctx context.Context, scope transaction.Scope, title string, dueDate time.Time) (domain.Todo, error)
}

// CreatorImpl is the implementation of the Creator use case.
type CreatorImpl struct {
	timeProvider core.CurrentTimeProvider
	createUUID   func() uuid.UUID
	encoder      semantic.Encoder
	llmModel     string
}

// NewCreatorImpl creates a new instance of CreatorImpl.
func NewCreatorImpl(timeProvider core.CurrentTimeProvider, encoder semantic.Encoder, llmModel string) CreatorImpl {
	return CreatorImpl{
		timeProvider: timeProvider,
		createUUID:   uuid.New,
		encoder:      encoder,
		llmModel:     llmModel,
	}
}

// Create creates a new todo item within the provided unit of work scope.
func (tci CreatorImpl) Create(ctx context.Context, scope transaction.Scope, title string, dueDate time.Time) (domain.Todo, error) {
	now := tci.timeProvider.Now()

	todo := domain.Todo{
		ID:        tci.createUUID(),
		Title:     title,
		Status:    domain.Status_OPEN,
		DueDate:   dueDate.UTC(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := todo.Validate(now); err != nil {
		return domain.Todo{}, err
	}

	resp, err := tci.encoder.VectorizeTodo(ctx, tci.llmModel, todo)
	if err != nil {
		return domain.Todo{}, err
	}

	metrics.RecordLLMTokensEmbedding(ctx, resp.TotalTokens)
	todo.Embedding = resp.Vector

	err = scope.Todo().CreateTodo(ctx, todo)
	if err != nil {
		return domain.Todo{}, err
	}

	err = scope.Outbox().CreateTodoEvent(ctx, outbox.TodoEvent{
		Type:      outbox.EventType_TODO_CREATED,
		TodoID:    todo.ID,
		CreatedAt: now,
	})
	if err != nil {
		return domain.Todo{}, err
	}

	return todo, nil
}
