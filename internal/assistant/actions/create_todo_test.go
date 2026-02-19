package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoCreatorAction(t *testing.T) {
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setupMocks func(
			*domain.MockUnitOfWork,
			*domain.MockCurrentTimeProvider,
			*usecases.MockTodoCreator,
		)
		functionCall domain.AssistantActionCall
		history      []domain.AssistantMessage
		validateResp func(t *testing.T, resp domain.AssistantMessage)
	}{
		"create-todo-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				creator.EXPECT().
					Create(mock.Anything, uow, "New Todo", mock.Anything).
					Return(domain.Todo{Title: "New Todo", Status: domain.TodoStatus_OPEN}, nil).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "create_todo",
				Input: `{"title": "New Todo", "due_date": "2026-01-25"}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "todos[1]{id,title,due_date,status}")
			},
		},
		"create-todo-empty-due-date-uses-history": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				creator.EXPECT().
					Create(mock.Anything, uow, "New Todo", mock.Anything).
					Return(domain.Todo{Title: "New Todo"}, nil).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "create_todo",
				Input: `{"title": "New Todo", "due_date": ""}`,
			},
			history: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "Please set it for tomorrow"},
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "todos[1]{id,title,due_date,status}")
			},
		},
		"create-todo-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "create_todo",
				Input: `invalid json`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"create-todo-invalid-due-date": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "create_todo",
				Input: `{"title": "New Todo", "due_date": "invalid"}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_date")
			},
		},
		"create-todo-create-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				creator.EXPECT().
					Create(mock.Anything, uow, "New Todo", mock.Anything).
					Return(domain.Todo{}, errors.New("create error")).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "create_todo",
				Input: `{"title": "New Todo", "due_date": "2026-01-25"}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "create_todo_error")
			},
		},
		"create-todo-uow-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					Return(errors.New("uow error")).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "create_todo",
				Input: `{"title": "New Todo", "due_date": "2026-01-25"}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "create_todo_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			todoCreator := usecases.NewMockTodoCreator(t)
			tt.setupMocks(uow, timeProvider, todoCreator)

			action := NewTodoCreatorAction(uow, todoCreator, timeProvider)
			assert.NotEmpty(t, action.StatusMessage())

			definition := action.Definition()
			assert.Equal(t, "create_todo", definition.Name)
			assert.NotEmpty(t, definition.Description)
			assert.NotEmpty(t, definition.Input)

			resp := action.Execute(context.Background(), tt.functionCall, tt.history)
			tt.validateResp(t, resp)
		})
	}
}
