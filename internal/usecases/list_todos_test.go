package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListTodosImpl_Query(t *testing.T) {
	tests := map[string]struct {
		setExpectations func(repo *domain.MockTodoRepository)
		page            int
		pageSize        int
		expectedTodos   []domain.Todo
		expectedHasMore bool
		expectedErr     error
	}{
		"success": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *domain.MockTodoRepository) {
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
		"repository-error": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *domain.MockTodoRepository) {
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
			if tt.setExpectations != nil {
				tt.setExpectations(repo)
			}

			lti := NewListTodosImpl(repo)

			got, hasMore, gotErr := lti.Query(context.Background(), tt.page, tt.pageSize)
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
