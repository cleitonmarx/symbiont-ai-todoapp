package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/toon-format/toon-go"
)

func TestUpdateTodosAction(t *testing.T) {
	t.Parallel()

	todoID1 := uuid.New()
	todoID2 := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*transaction.MockUnitOfWork, *todouc.MockUpdater)
		functionCall assistant.ActionCall
		validateResp func(t *testing.T, resp assistant.Message)
	}{
		"update-todos-success": {
			setupMocks: func(uow *transaction.MockUnitOfWork, updater *todouc.MockUpdater) {
				scope := transaction.NewMockScope(t)

				updater.EXPECT().
					Update(
						mock.Anything,
						scope,
						todoID1,
						common.Ptr("Updated 1"),
						common.Ptr(todo.Status_DONE),
						(*time.Time)(nil),
					).
					Return(
						todo.Todo{
							ID:     todoID1,
							Title:  "Updated 1",
							Status: todo.Status_DONE,
						},
						nil,
					).
					Once()
				updater.EXPECT().
					Update(
						mock.Anything,
						scope,
						todoID2,
						common.Ptr("Updated 2"),
						(*todo.Status)(nil),
						(*time.Time)(nil),
					).
					Return(
						todo.Todo{
							ID:     todoID2,
							Title:  "Updated 2",
							Status: todo.Status_OPEN,
						},
						nil,
					).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					}).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "update_todos",
				Input: `{"todos":[{"id":"` + todoID1.String() + `","title":"Updated 1","status":"DONE"},{"id":"` + todoID2.String() + `","title":"Updated 2"}]}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				payload := struct {
					Todos []struct {
						Title string `toon:"title"`
					} `toon:"todos"`
				}{}
				assert.NoError(t, toon.UnmarshalString(resp.Content, &payload))
				assert.Len(t, payload.Todos, 2)
			},
		},
		"update-todos-invalid-arguments": {
			setupMocks: func(uow *transaction.MockUnitOfWork, updater *todouc.MockUpdater) {
			},
			functionCall: assistant.ActionCall{
				Name:  "update_todos",
				Input: `invalid json`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"update-todos-invalid-status": {
			setupMocks: func(uow *transaction.MockUnitOfWork, updater *todouc.MockUpdater) {
			},
			functionCall: assistant.ActionCall{
				Name:  "update_todos",
				Input: `{"todos":[{"id":"` + todoID1.String() + `","status":"INVALID"}]}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Contains(t, resp.Content, "invalid_status")
			},
		},
		"update-todos-update-error": {
			setupMocks: func(uow *transaction.MockUnitOfWork, updater *todouc.MockUpdater) {
				scope := transaction.NewMockScope(t)

				updater.EXPECT().
					Update(
						mock.Anything,
						scope,
						todoID1,
						common.Ptr("Updated 1"),
						(*todo.Status)(nil),
						(*time.Time)(nil),
					).
					Return(todo.Todo{}, errors.New("update error")).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					}).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "update_todos",
				Input: `{"todos":[{"id":"` + todoID1.String() + `","title":"Updated 1"}]}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Contains(t, resp.Content, "update_todos_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := transaction.NewMockUnitOfWork(t)
			updater := todouc.NewMockUpdater(t)
			tt.setupMocks(uow, updater)

			action := NewUpdateTodosAction(uow, updater)
			assert.NotEmpty(t, action.StatusMessage())

			definition := action.Definition()
			assert.Equal(t, "update_todos", definition.Name)
			assert.NotEmpty(t, definition.Description)
			assert.NotEmpty(t, definition.Input)
			assert.True(t, definition.Approval.Required)
			assert.Equal(t, "Confirm update of todos", definition.Approval.Title)
			assert.Equal(t, "Updating todos will modify existing items. Please confirm.", definition.Approval.Description)
			assert.Equal(t, []string{"todos[].id", "todos[].title", "todos[].status"}, definition.Approval.PreviewFields)
			assert.Equal(t, 2*time.Minute, definition.Approval.Timeout)

			resp := action.Execute(t.Context(), tt.functionCall, []assistant.Message{})
			tt.validateResp(t, resp)
		})
	}
}
