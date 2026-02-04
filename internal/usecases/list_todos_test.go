package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListTodosImpl_Query(t *testing.T) {
	tests := map[string]struct {
		setExpectations func(repo *domain.MockTodoRepository, llmClient *domain.MockLLMClient)
		page            int
		pageSize        int
		queryParams     []ListTodoOptions
		expectedTodos   []domain.Todo
		expectedHasMore bool
		expectedErr     error
	}{
		"success": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *domain.MockTodoRepository, llmClient *domain.MockLLMClient) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).Return([]domain.Todo{
					{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Title: "Todo 1", Status: domain.TodoStatus_OPEN},
					{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"), Title: "Todo 2", Status: domain.TodoStatus_OPEN},
				}, true, nil)
			},
			expectedTodos: []domain.Todo{
				{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Title: "Todo 1", Status: domain.TodoStatus_OPEN},
				{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"), Title: "Todo 2", Status: domain.TodoStatus_OPEN},
			},
			expectedHasMore: true,
			expectedErr:     nil,
		},
		"success-with-status-filter": {
			page:     1,
			pageSize: 10,
			queryParams: []ListTodoOptions{
				WithStatus(domain.TodoStatus_DONE),
			},
			setExpectations: func(repo *domain.MockTodoRepository, llmClient *domain.MockLLMClient) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).
					Run(func(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) {
						var params domain.ListTodosParams
						for _, opt := range opts {
							opt(&params)
						}
						assert.Equal(t, domain.TodoStatus_DONE, *params.Status)
					}).
					Return([]domain.Todo{
						{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"), Title: "Todo 3", Status: domain.TodoStatus_DONE},
					}, false, nil)
			},
			expectedTodos: []domain.Todo{
				{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"), Title: "Todo 3", Status: domain.TodoStatus_DONE},
			},
			expectedHasMore: false,
			expectedErr:     nil,
		},
		"success-without-query-filters": {
			page:     2,
			pageSize: 5,
			queryParams: []ListTodoOptions{
				WithSearchQuery("meeting"),
			},
			setExpectations: func(repo *domain.MockTodoRepository, llmClient *domain.MockLLMClient) {
				llmClient.EXPECT().
					Embed(mock.Anything, "test-model", "meeting").
					Return([]float64{0.1, 0.2, 0.3}, nil)

				repo.EXPECT().ListTodos(mock.Anything, 2, 5, mock.Anything).
					Run(func(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) {
						var params domain.ListTodosParams
						for _, opt := range opts {
							opt(&params)
						}
						assert.Equal(t, []float64{0.1, 0.2, 0.3}, params.Embedding) // This line seems incorrect; should check SearchQuery instead --- IGNORE ---
					}).
					Return([]domain.Todo{}, false, nil)
			},
			expectedTodos:   []domain.Todo{},
			expectedHasMore: false,
			expectedErr:     nil,
		},
		"success-with-due-date-range-filter": {
			page:     1,
			pageSize: 10,
			queryParams: []ListTodoOptions{
				WithDueDateRange(
					time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
				),
			},
			setExpectations: func(repo *domain.MockTodoRepository, llmClient *domain.MockLLMClient) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).
					Run(func(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) {
						var params domain.ListTodosParams
						for _, opt := range opts {
							opt(&params)
						}
						assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), *params.DueAfter)
						assert.Equal(t, time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), *params.DueBefore)
					}).
					Return([]domain.Todo{}, false, nil)
			},
			expectedTodos:   []domain.Todo{},
			expectedHasMore: false,
			expectedErr:     nil,
		},
		"sort-by-created-at-desc": {
			page:     1,
			pageSize: 10,
			queryParams: []ListTodoOptions{
				WithSortBy("createdAtDesc"),
			},
			setExpectations: func(repo *domain.MockTodoRepository, llmClient *domain.MockLLMClient) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).
					Run(func(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) {
						var params domain.ListTodosParams
						for _, opt := range opts {
							opt(&params)
						}
						assert.Equal(t, "createdAt", params.SortBy.Field)
						assert.Equal(t, "DESC", params.SortBy.Direction)
					}).
					Return([]domain.Todo{}, false, nil)
			},
			expectedTodos:   []domain.Todo{},
			expectedHasMore: false,
			expectedErr:     nil,
		},
		"repository-error": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *domain.MockTodoRepository, llmClient *domain.MockLLMClient) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).Return(nil, false, errors.New("database error"))
			},
			expectedTodos:   nil,
			expectedHasMore: false,
			expectedErr:     errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain.NewMockTodoRepository(t)
			llmCli := domain.NewMockLLMClient(t)
			if tt.setExpectations != nil {
				tt.setExpectations(repo, llmCli)
			}

			lti := NewListTodosImpl(repo, llmCli, "test-model")

			got, hasMore, gotErr := lti.Query(context.Background(), tt.page, tt.pageSize, tt.queryParams...)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.Equal(t, tt.expectedTodos, got)
			assert.Equal(t, tt.expectedHasMore, hasMore)
		})
	}
}

func TestInitListTodos_Initialize(t *testing.T) {
	ilt := InitListTodos{}

	ctx, err := ilt.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredListTodos, err := depend.Resolve[ListTodos]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredListTodos)
}
