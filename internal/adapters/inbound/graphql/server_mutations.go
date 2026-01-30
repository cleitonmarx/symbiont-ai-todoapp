package graphql

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/types"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
)

// Mutation returns MutationResolver implementation.
func (s *TodoGraphQLServer) Mutation() gen.MutationResolver { return s }

// UpdateTodo is the resolver for the updateTodo field.
func (s *TodoGraphQLServer) UpdateTodo(ctx context.Context, params gen.UpdateTodoParams) (*gen.Todo, error) {
	td, err := s.UpdateTodoUsecase.Execute(
		ctx,
		params.ID,
		params.Title,
		(*domain.TodoStatus)(params.Status),
		(*time.Time)(params.DueDate),
	)
	return &gen.Todo{
		ID:        td.ID,
		Title:     td.Title,
		Status:    gen.TodoStatus(td.Status),
		DueDate:   (types.Date)(td.DueDate),
		CreatedAt: td.CreatedAt,
		UpdatedAt: td.UpdatedAt,
	}, err
}

// DeleteTodo is the resolver for the deleteTodo field.
func (s *TodoGraphQLServer) DeleteTodo(ctx context.Context, id uuid.UUID) (bool, error) {
	err := s.DeleteTodoUsecase.Execute(ctx, id)
	if err != nil {
		return false, err
	}
	return true, nil
}
