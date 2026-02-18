package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoUpdaterAction(t *testing.T) {
	todoID := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*domain.MockUnitOfWork, *usecases.MockTodoUpdater)
		functionCall domain.AssistantActionCall
		validateResp func(t *testing.T, resp domain.AssistantMessage)
	}{
		"update-todo-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, updater *usecases.MockTodoUpdater) {
				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						common.Ptr("Updated"),
						common.Ptr(domain.TodoStatus_DONE),
						(*time.Time)(nil),
					).
					Return(
						domain.Todo{
							ID:     todoID,
							Title:  "Updated",
							Status: domain.TodoStatus_DONE,
						},
						nil,
					)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "update_todo",
				Input: `{"id": "` + todoID.String() + `", "title": "Updated", "status": "DONE"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "updated successfully")
			},
		},
		"update-todo-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, updater *usecases.MockTodoUpdater) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "update_todo",
				Input: `invalid json`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"update-todo-invalid-id": {
			setupMocks: func(uow *domain.MockUnitOfWork, updater *usecases.MockTodoUpdater) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "update_todo",
				Input: `{"id": "invalid-uuid", "title": "Updated"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_todo_id")
			},
		},
		"update-todo-update-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, updater *usecases.MockTodoUpdater) {
				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						common.Ptr("Updated"),
						(*domain.TodoStatus)(nil),
						(*time.Time)(nil),
					).
					Return(domain.Todo{}, errors.New("update error"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "update_todo",
				Input: `{"id": "` + todoID.String() + `", "title": "Updated"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "update_todo_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			updater := usecases.NewMockTodoUpdater(t)
			tt.setupMocks(uow, updater)

			action := NewTodoUpdaterAction(uow, updater)

			resp := action.Execute(context.Background(), tt.functionCall, []domain.AssistantMessage{})
			tt.validateResp(t, resp)
		})
	}
}
