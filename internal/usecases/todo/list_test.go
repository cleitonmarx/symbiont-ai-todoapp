package todo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListImpl_Query(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setExpectations func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder)
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
			setExpectations: func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).Return([]domain.Todo{
					{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Title: "Todo 1", Status: domain.Status_OPEN},
					{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"), Title: "Todo 2", Status: domain.Status_OPEN},
				}, true, nil)
			},
			expectedTodos: []domain.Todo{
				{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Title: "Todo 1", Status: domain.Status_OPEN},
				{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"), Title: "Todo 2", Status: domain.Status_OPEN},
			},
			expectedHasMore: true,
			expectedErr:     nil,
		},
		"success-with-status-filter": {
			page:     1,
			pageSize: 10,
			queryParams: []ListTodoOptions{
				WithStatus(domain.Status_DONE),
			},
			setExpectations: func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).
					Run(func(ctx context.Context, page int, pageSize int, opts ...domain.ListOption) {
						var params domain.ListParams
						for _, opt := range opts {
							opt(&params)
						}
						assert.Equal(t, domain.Status_DONE, *params.Status)
					}).
					Return([]domain.Todo{
						{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"), Title: "Todo 3", Status: domain.Status_DONE},
					}, false, nil)
			},
			expectedTodos: []domain.Todo{
				{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"), Title: "Todo 3", Status: domain.Status_DONE},
			},
			expectedHasMore: false,
			expectedErr:     nil,
		},
		"success-search-similarity-filters": {
			page:     2,
			pageSize: 5,
			queryParams: []ListTodoOptions{
				WithSearchQuery("meeting"),
				WithSearchType(SearchType_Similarity),
			},
			setExpectations: func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder) {
				semanticEncoder.EXPECT().
					VectorizeQuery(mock.Anything, "test-model", "meeting").
					Return(semantic.EmbeddingVector{Vector: []float64{0.1, 0.2, 0.3}}, nil)

				repo.EXPECT().ListTodos(mock.Anything, 2, 5, mock.Anything).
					Run(func(ctx context.Context, page int, pageSize int, opts ...domain.ListOption) {
						var params domain.ListParams
						for _, opt := range opts {
							opt(&params)
						}
						assert.Equal(t, []float64{0.1, 0.2, 0.3}, params.Embedding)
					}).
					Return([]domain.Todo{}, false, nil)
			},
			expectedTodos:   []domain.Todo{},
			expectedHasMore: false,
			expectedErr:     nil,
		},
		"success-search-title-filter": {
			page:     1,
			pageSize: 5,
			queryParams: []ListTodoOptions{
				WithSearchQuery("report"),
				WithSearchType(SearchType_Title),
			},
			setExpectations: func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 5, mock.Anything).
					Run(func(ctx context.Context, page int, pageSize int, opts ...domain.ListOption) {
						var params domain.ListParams
						for _, opt := range opts {
							opt(&params)
						}
						assert.Equal(t, "report", *params.TitleContains)
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
			setExpectations: func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).
					Run(func(ctx context.Context, page int, pageSize int, opts ...domain.ListOption) {
						var params domain.ListParams
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
			setExpectations: func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).
					Run(func(ctx context.Context, page int, pageSize int, opts ...domain.ListOption) {
						var params domain.ListParams
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
		"error-when-due-range-is-invalid": {
			page:     1,
			pageSize: 10,
			queryParams: []ListTodoOptions{
				WithDueDateRange(
					time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
					time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				),
			},
			setExpectations: func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder) {
			},
			expectedTodos:   []domain.Todo(nil),
			expectedHasMore: false,
			expectedErr:     core.NewValidationErr("due_after must be less than or equal to due_before"),
		},
		"error-when-search-type-is-not-provided": {
			page:     1,
			pageSize: 5,
			queryParams: []ListTodoOptions{
				WithSearchQuery("meeting"),
			},
			setExpectations: func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder) {
			},
			expectedTodos:   []domain.Todo(nil),
			expectedHasMore: false,
			expectedErr:     core.NewValidationErr("invalid search type"),
		},
		"repository-error": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *domain.MockRepository, semanticEncoder *semantic.MockEncoder) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 10, mock.Anything).Return(nil, false, errors.New("database error"))
			},
			expectedTodos:   nil,
			expectedHasMore: false,
			expectedErr:     errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain.NewMockRepository(t)
			semanticEncoder := semantic.NewMockEncoder(t)
			if tt.setExpectations != nil {
				tt.setExpectations(repo, semanticEncoder)
			}

			lti := NewListImpl(repo, semanticEncoder, "test-model")

			got, hasMore, gotErr := lti.Query(context.Background(), tt.page, tt.pageSize, tt.queryParams...)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.Equal(t, tt.expectedTodos, got)
			assert.Equal(t, tt.expectedHasMore, hasMore)
		})
	}
}
