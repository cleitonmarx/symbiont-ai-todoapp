package todo

import (
	"context"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/metrics"
	"github.com/google/uuid"
)

// Updater defines the interface for modifying todo items.
type Updater interface {
	Update(ctx context.Context, scope transaction.Scope, id uuid.UUID, title *string, status *domain.Status, dueDate *time.Time) (domain.Todo, error)
}

// UpdaterImpl is the implementation of the Updater interface.
type UpdaterImpl struct {
	timeProvider core.CurrentTimeProvider
	encoder      semantic.Encoder
	model        string
}

// NewUpdaterImpl creates a new instance of UpdaterImpl.
func NewUpdaterImpl(
	timeProvider core.CurrentTimeProvider,
	encoder semantic.Encoder,
	model string,
) UpdaterImpl {
	return UpdaterImpl{
		timeProvider: timeProvider,
		encoder:      encoder,
		model:        model,
	}
}

// Update modifies an existing todo item identified by id with the provided title, status, and/or due date.
func (tui UpdaterImpl) Update(ctx context.Context, scope transaction.Scope, id uuid.UUID, title *string, status *domain.Status, dueDate *time.Time) (domain.Todo, error) {
	now := tui.timeProvider.Now()
	var todo domain.Todo
	td, found, err := scope.Todo().GetTodo(ctx, id)
	if err != nil {
		return domain.Todo{}, err
	}
	if !found {
		return domain.Todo{}, core.NewNotFoundErr(fmt.Sprintf("todo with ID %s not found", id))
	}

	if title != nil {
		td.Title = *title
	}

	if status != nil {
		td.Status = *status
	}

	if dueDate != nil {
		td.DueDate = dueDate.UTC()
	}

	td.UpdatedAt = now

	if err := td.Validate(now); err != nil {
		return domain.Todo{}, err
	}

	resp, err := tui.encoder.VectorizeTodo(ctx, tui.model, td)
	if err != nil {
		return domain.Todo{}, err
	}

	metrics.RecordLLMTokensEmbedding(ctx, resp.TotalTokens)
	td.Embedding = resp.Vector

	if err := scope.Todo().UpdateTodo(ctx, td); err != nil {
		return domain.Todo{}, err
	}

	todo = td

	if err = scope.Outbox().CreateTodoEvent(ctx, outbox.TodoEvent{
		Type:      outbox.EventType_TODO_UPDATED,
		TodoID:    todo.ID,
		CreatedAt: now,
	}); err != nil {
		return domain.Todo{}, err
	}

	return todo, nil
}
