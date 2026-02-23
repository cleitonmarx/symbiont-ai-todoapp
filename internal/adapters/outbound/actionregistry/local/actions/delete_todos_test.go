package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBulkTodoDeleterAction(t *testing.T) {
	t.Parallel()

	todoID1 := uuid.New()
	todoID2 := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*domain.MockUnitOfWork, *usecases.MockTodoDeleter)
		functionCall domain.AssistantActionCall
		validateResp func(t *testing.T, resp domain.AssistantMessage)
	}{
		"delete-todos-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *usecases.MockTodoDeleter) {
				deleter.EXPECT().
					Delete(mock.Anything, uow, todoID1).
					Return(nil).
					Once()
				deleter.EXPECT().
					Delete(mock.Anything, uow, todoID2).
					Return(nil).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "delete_todos",
				Input: `{"ids":["` + todoID1.String() + `","` + todoID2.String() + `"]}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "todos[2]{id,deleted}")
			},
		},
		"delete-todos-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *usecases.MockTodoDeleter) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "delete_todos",
				Input: `invalid json`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"delete-todos-invalid-id": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *usecases.MockTodoDeleter) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "delete_todos",
				Input: `{"ids":["invalid-uuid"]}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "invalid_todo_id")
			},
		},
		"delete-todos-delete-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *usecases.MockTodoDeleter) {
				deleter.EXPECT().
					Delete(mock.Anything, uow, todoID1).
					Return(errors.New("delete error")).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "delete_todos",
				Input: `{"ids":["` + todoID1.String() + `"]}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "delete_todos_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			deleter := usecases.NewMockTodoDeleter(t)
			tt.setupMocks(uow, deleter)

			action := NewBulkTodoDeleterAction(uow, deleter)
			assert.NotEmpty(t, action.StatusMessage())

			definition := action.Definition()
			assert.Equal(t, "delete_todos", definition.Name)
			assert.NotEmpty(t, definition.Description)
			assert.NotEmpty(t, definition.Input)

			resp := action.Execute(context.Background(), tt.functionCall, []domain.AssistantMessage{})
			tt.validateResp(t, resp)
		})
	}
}
