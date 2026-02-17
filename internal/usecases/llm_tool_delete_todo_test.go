package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoDeleterTool(t *testing.T) {
	todoID := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*domain.MockUnitOfWork, *MockTodoDeleter)
		functionCall domain.LLMStreamEventToolCall
		validateResp func(t *testing.T, resp domain.LLMChatMessage)
	}{
		"delete-todo-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *MockTodoDeleter) {
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
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "delete_todo",
				Arguments: `{"id": "` + todoID.String() + `"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "deleted successfully")
			},
		},
		"delete-todo-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *MockTodoDeleter) {
			},
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "delete_todo",
				Arguments: `invalid json`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"delete-todo-delete-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *MockTodoDeleter) {
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
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "delete_todo",
				Arguments: `{"id": "` + todoID.String() + `"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "delete_todo_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			deleter := NewMockTodoDeleter(t)
			tt.setupMocks(uow, deleter)

			tool := NewTodoDeleterTool(uow, deleter)

			resp := tool.Call(context.Background(), tt.functionCall, []domain.LLMChatMessage{})
			tt.validateResp(t, resp)
		})
	}
}
