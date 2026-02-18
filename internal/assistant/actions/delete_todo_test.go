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

func TestTodoDeleterAction(t *testing.T) {
	todoID := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*domain.MockUnitOfWork, *usecases.MockTodoDeleter)
		functionCall domain.AssistantActionCall
		validateResp func(t *testing.T, resp domain.AssistantMessage)
	}{
		"delete-todo-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *usecases.MockTodoDeleter) {
				deleter.EXPECT().
					Delete(
						mock.Anything,
						uow,
						todoID,
					).
					Return(nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "delete_todo",
				Input: `{"id": "` + todoID.String() + `"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "deleted successfully")
			},
		},
		"delete-todo-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *usecases.MockTodoDeleter) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "delete_todo",
				Input: `invalid json`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"delete-todo-delete-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *usecases.MockTodoDeleter) {
				deleter.EXPECT().
					Delete(
						mock.Anything,
						uow,
						todoID,
					).
					Return(errors.New("delete error"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "delete_todo",
				Input: `{"id": "` + todoID.String() + `"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "delete_todo_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			deleter := usecases.NewMockTodoDeleter(t)
			tt.setupMocks(uow, deleter)

			tool := NewTodoDeleterAction(uow, deleter)

			resp := tool.Execute(context.Background(), tt.functionCall, []domain.AssistantMessage{})
			tt.validateResp(t, resp)
		})
	}
}
