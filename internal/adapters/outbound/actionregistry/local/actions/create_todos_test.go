package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/toon-format/toon-go"
)

func TestCreateTodosAction(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setupMocks func(
			*transaction.MockUnitOfWork,
			*core.MockCurrentTimeProvider,
			*todouc.MockCreator,
		)
		functionCall assistant.ActionCall
		history      []assistant.Message
		validateResp func(t *testing.T, resp assistant.Message)
	}{
		"create-todos-success": {
			setupMocks: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider, creator *todouc.MockCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				scope := transaction.NewMockScope(t)

				creator.EXPECT().
					Create(mock.Anything, scope, "Todo 1", mock.Anything).
					Return(todo.Todo{
						ID:      uuid.New(),
						Title:   "Todo 1",
						DueDate: time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC),
						Status:  todo.Status_OPEN,
					}, nil).
					Once()
				creator.EXPECT().
					Create(mock.Anything, scope, "Todo 2", mock.Anything).
					Return(todo.Todo{
						ID:      uuid.New(),
						Title:   "Todo 2",
						DueDate: time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC),
						Status:  todo.Status_OPEN,
					}, nil).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					}).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "create_todos",
				Input: `{"todos":[{"title":"Todo 1","due_date":"2026-01-25"},{"title":"Todo 2","due_date":"2026-01-26"}]}`,
			},
			history: []assistant.Message{},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				payload := struct {
					Todos []struct {
						Title string `toon:"title"`
					} `toon:"todos"`
				}{}
				assert.NoError(t, toon.UnmarshalString(resp.Content, &payload))
				assert.Len(t, payload.Todos, 2)
			},
		},
		"create-todos-invalid-arguments": {
			setupMocks: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider, creator *todouc.MockCreator) {
			},
			functionCall: assistant.ActionCall{
				Name:  "create_todos",
				Input: `invalid json`,
			},
			history: []assistant.Message{},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"create-todos-invalid-due-date": {
			setupMocks: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider, creator *todouc.MockCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "create_todos",
				Input: `{"todos":[{"title":"Todo 1","due_date":"invalid"}]}`,
			},
			history: []assistant.Message{},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Contains(t, resp.Content, "invalid_due_date")
			},
		},
		"create-todos-create-error": {
			setupMocks: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider, creator *todouc.MockCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
				scope := transaction.NewMockScope(t)

				creator.EXPECT().
					Create(mock.Anything, scope, "Todo 1", mock.Anything).
					Return(todo.Todo{}, errors.New("create error")).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					}).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "create_todos",
				Input: `{"todos":[{"title":"Todo 1","due_date":"2026-01-25"}]}`,
			},
			history: []assistant.Message{},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Contains(t, resp.Content, "create_todos_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := transaction.NewMockUnitOfWork(t)
			timeProvider := core.NewMockCurrentTimeProvider(t)
			todoCreator := todouc.NewMockCreator(t)
			tt.setupMocks(uow, timeProvider, todoCreator)

			action := NewCreateTodosAction(uow, todoCreator, timeProvider)
			assert.NotEmpty(t, action.StatusMessage())

			definition := action.Definition()
			assert.Equal(t, "create_todos", definition.Name)
			assert.NotEmpty(t, definition.Description)
			assert.NotEmpty(t, definition.Input)

			resp := action.Execute(context.Background(), tt.functionCall, tt.history)
			tt.validateResp(t, resp)
		})
	}
}
