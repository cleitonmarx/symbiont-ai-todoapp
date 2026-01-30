package graphql

import (
	"context"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/types"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
)

// ListTodos is the resolver for the listTodos field.
func (s *TodoGraphQLServer) ListTodos(ctx context.Context, status *gen.TodoStatus, page int, pageSize int) (*gen.TodoPage, error) {
	var options []domain.ListTodoOptions
	if status != nil {
		options = append(options, domain.WithStatus(domain.TodoStatus(*status)))
	}
	todos, hasMore, err := s.ListTodosUsecase.Query(ctx, page, pageSize, options...)
	if err != nil {
		return nil, err
	}

	todoPage := gen.TodoPage{
		Items: make([]*gen.Todo, len(todos)),
		Page:  page,
	}

	for i, t := range todos {
		todoPage.Items[i] = &gen.Todo{
			ID:        t.ID,
			Title:     t.Title,
			Status:    gen.TodoStatus(t.Status),
			DueDate:   (types.Date)(t.DueDate),
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		}
	}

	if hasMore {
		todoPage.NextPage = common.Ptr(page + 1)
	}
	if page > 1 {
		todoPage.PreviousPage = common.Ptr(page - 1)
	}

	return &todoPage, nil
}

// Query returns QueryResolver implementation.
func (s *TodoGraphQLServer) Query() gen.QueryResolver { return s }
