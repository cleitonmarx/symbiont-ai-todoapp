package graphql

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql/types"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// Mutation returns MutationResolver implementation.
func (s *TodoGraphQLServer) Mutation() gen.MutationResolver { return s }

// UpdateTodo is the resolver for the updateTodo field.
func (s *TodoGraphQLServer) UpdateTodo(ctx context.Context, params gen.UpdateTodoParams) (*gen.Todo, error) {
	td, err := s.UpdateTodoUsecase.Execute(
		ctx,
		params.ID,
		params.Title,
		(*todo.Status)(params.Status),
		(*time.Time)(params.DueDate),
	)
	if telemetry.IsErrorRecorded(trace.SpanFromContext(ctx), err) {
		s.Logger.Printf("Error updating todo: %v", err)
		return nil, err
	}

	return &gen.Todo{
		ID:        td.ID,
		Title:     td.Title,
		Status:    gen.TodoStatus(td.Status),
		DueDate:   (types.Date)(td.DueDate),
		CreatedAt: td.CreatedAt,
		UpdatedAt: td.UpdatedAt,
	}, nil
}

// DeleteTodo is the resolver for the deleteTodo field.
func (s *TodoGraphQLServer) DeleteTodo(ctx context.Context, id uuid.UUID) (bool, error) {
	err := s.DeleteTodoUsecase.Execute(ctx, id)
	if telemetry.IsErrorRecorded(trace.SpanFromContext(ctx), err) {
		s.Logger.Printf("Error deleting todo: %v", err)
		return false, err
	}

	return true, nil
}
