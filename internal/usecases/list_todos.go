package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
)

// ListTodos defines the interface for the ListTodos use case.
type ListTodos interface {
	Query(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) ([]domain.Todo, bool, error)
}

// ListTodosImpl is the implementation of the ListTodos use case.
type ListTodosImpl struct {
	TodoRepo domain.TodoRepository `resolve:""`
}

// NewListTodosImpl creates a new instance of ListTodosImpl.
func NewListTodosImpl(todoRepo domain.TodoRepository) ListTodosImpl {
	return ListTodosImpl{
		TodoRepo: todoRepo,
	}
}

// Query retrieves a list of todo items with pagination support.
func (lti ListTodosImpl) Query(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) ([]domain.Todo, bool, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	todos, hasMore, err := lti.TodoRepo.ListTodos(spanCtx, page, pageSize, opts...)
	if tracing.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	return todos, hasMore, nil
}

// InitListTodos initializes the ListTodos use case and registers it in the dependency container.
type InitListTodos struct {
	TodoRepo domain.TodoRepository `resolve:""`
}

// Initialize initializes the ListTodosImpl use case and registers it in the dependency container.
func (ilt InitListTodos) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListTodos](NewListTodosImpl(ilt.TodoRepo))
	return ctx, nil
}
