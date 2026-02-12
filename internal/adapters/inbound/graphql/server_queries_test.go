package graphql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/types"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoGraphQLServer_ListTodos(t *testing.T) {
	tests := map[string]struct {
		page          int
		pageSize      int
		status        *gen.TodoStatus
		query         *string
		dateRange     *gen.DateRange
		sortBy        *gen.TodoSortBy
		setupUsecases func(*usecases.MockListTodos)
		expected      *gen.TodoPage
		expectError   bool
	}{
		"success": {
			status:   (*gen.TodoStatus)(&testStatus),
			page:     2,
			pageSize: 1,
			setupUsecases: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 2, 1, mock.Anything).
					Return([]domain.Todo{testTodo}, true, nil)
			},
			expected: &gen.TodoPage{
				Items:        []*gen.Todo{&testGenTodo},
				Page:         2,
				NextPage:     common.Ptr(3),
				PreviousPage: common.Ptr(1),
			},
			expectError: false,
		},
		"success-with-query": {
			status:   nil,
			page:     1,
			pageSize: 2,
			query:    common.Ptr("groceries"),
			setupUsecases: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 2, mock.Anything).
					Run(func(_ context.Context, _ int, _ int, opts ...usecases.ListTodoOptions) {
						p := usecases.ListTodoParams{}
						for _, opt := range opts {
							opt(&p)
						}
						assert.NotNil(t, p.Query)
						assert.Equal(t, "groceries", *p.Query)
					}).
					Return([]domain.Todo{testTodo}, false, nil)
			},
			expected: &gen.TodoPage{
				Items: []*gen.Todo{&testGenTodo},
				Page:  1,
			},
			expectError: false,
		},
		"success-with-date-range": {
			status:   nil,
			page:     1,
			pageSize: 2,
			dateRange: &gen.DateRange{
				DueAfter:  types.Date(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				DueBefore: types.Date(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)),
			},
			setupUsecases: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 2, mock.Anything).
					Run(func(_ context.Context, _ int, _ int, opts ...usecases.ListTodoOptions) {
						p := usecases.ListTodoParams{}
						for _, opt := range opts {
							opt(&p)
						}
						assert.NotNil(t, p.DueAfter)
						assert.NotNil(t, p.DueBefore)
						assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), *p.DueAfter)
						assert.Equal(t, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), *p.DueBefore)
					}).
					Return([]domain.Todo{testTodo}, false, nil)
			},
			expected: &gen.TodoPage{
				Items: []*gen.Todo{&testGenTodo},
				Page:  1,
			},
			expectError: false,
		},
		"success-with-sort-by": {
			status:   nil,
			page:     1,
			pageSize: 2,
			sortBy:   (*gen.TodoSortBy)(common.Ptr(gen.TodoSortByDueDateDesc)),
			setupUsecases: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 2, mock.Anything).
					Run(func(_ context.Context, _ int, _ int, opts ...usecases.ListTodoOptions) {
						p := usecases.ListTodoParams{}
						for _, opt := range opts {
							opt(&p)
						}

						assert.Equal(t, p.SortBy, common.Ptr("dueDateDesc"))
					}).
					Return([]domain.Todo{testTodo}, false, nil)
			},
			expected: &gen.TodoPage{
				Items: []*gen.Todo{&testGenTodo},
				Page:  1,
			},
			expectError: false,
		},
		"error": {
			status:   nil,
			page:     1,
			pageSize: 2,
			setupUsecases: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 2, mock.Anything).
					Return(nil, false, errors.New("fail"))
			},
			expected:    nil,
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUC := usecases.NewMockListTodos(t)
			tt.setupUsecases(mockUC)
			server := &TodoGraphQLServer{ListTodosUsecase: mockUC}

			got, err := server.ListTodos(context.Background(), tt.page, tt.pageSize, tt.status, tt.query, tt.dateRange, tt.sortBy)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
			mockUC.AssertExpectations(t)
		})
	}
}
