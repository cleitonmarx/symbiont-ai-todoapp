package graphql

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql/types"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
)

// ListTodos is the resolver for the listTodos field.
func (s *TodoGraphQLServer) ListTodos(ctx context.Context, page int, pageSize int, status *gen.TodoStatus, search *string, searchType *gen.SearchType, dateRange *gen.DateRange, sortBy *gen.TodoSortBy) (*gen.TodoPage, error) {
	var options []todouc.ListTodoOptions
	if status != nil {
		options = append(options, todouc.WithStatus(todo.Status(*status)))
	}
	if search != nil {
		options = append(options, todouc.WithSearchQuery(*search))
	}
	if searchType != nil {
		options = append(options, todouc.WithSearchType(todouc.SearchType(*searchType)))
	}
	if dateRange != nil {
		options = append(options, todouc.WithDueDateRange(time.Time(dateRange.DueAfter), time.Time(dateRange.DueBefore)))
	}
	if sortBy != nil {
		options = append(options, todouc.WithSortBy(string(*sortBy)))
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
