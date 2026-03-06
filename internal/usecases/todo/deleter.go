package todo

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
)

// Deleter defines the interface for deleting todos within a unit of work scope.
type Deleter interface {
	Delete(ctx context.Context, scope transaction.Scope, id uuid.UUID) error
}

// DeleterImpl is the implementation of the Deleter interface.
type DeleterImpl struct {
	timeProvider core.CurrentTimeProvider
}

// NewDeleterImpl creates a new instance of DeleterImpl.
func NewDeleterImpl(timeProvider core.CurrentTimeProvider) DeleterImpl {
	return DeleterImpl{
		timeProvider: timeProvider,
	}
}

// Delete deletes a todo item by its ID.
func (dt DeleterImpl) Delete(ctx context.Context, scope transaction.Scope, id uuid.UUID) error {
	_, found, err := scope.Todo().GetTodo(ctx, id) // Ensure the todo exists
	if err != nil {
		return err
	}
	if !found {
		return core.NewNotFoundErr(fmt.Sprintf("todo with ID %s not found", id))
	}
	err = scope.Todo().DeleteTodo(ctx, id)
	if err != nil {
		return err
	}

	return scope.Outbox().CreateTodoEvent(ctx, outbox.TodoEvent{
		Type:      outbox.EventType_TODO_DELETED,
		TodoID:    id,
		CreatedAt: dt.timeProvider.Now(),
	})
}
