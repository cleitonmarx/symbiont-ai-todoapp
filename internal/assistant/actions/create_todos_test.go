package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBulkTodoCreatorAction(t *testing.T) {
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
		"create-todos-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				creator.EXPECT().
					Create(mock.Anything, uow, "Todo 1", mock.Anything).
					Return(domain.Todo{
						ID:      uuid.New(),
						Title:   "Todo 1",
						DueDate: time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC),
						Status:  domain.TodoStatus_OPEN,
					}, nil).
					Once()
				creator.EXPECT().
					Create(mock.Anything, uow, "Todo 2", mock.Anything).
					Return(domain.Todo{
						ID:      uuid.New(),
						Title:   "Todo 2",
						DueDate: time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC),
						Status:  domain.TodoStatus_OPEN,
					}, nil).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "create_todos",
				Input: `{"todos":[{"title":"Todo 1","due_date":"2026-01-25"},{"title":"Todo 2","due_date":"2026-01-26"}]}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "todos[2]{id,title,due_date,status}")
			},
		},
		"create-todos-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "create_todos",
				Input: `invalid json`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"create-todos-invalid-due-date": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "create_todos",
				Input: `{"todos":[{"title":"Todo 1","due_date":"invalid"}]}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "invalid_due_date")
			},
		},
		"create-todos-create-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *usecases.MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				creator.EXPECT().
					Create(mock.Anything, uow, "Todo 1", mock.Anything).
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
				Name:  "create_todos",
				Input: `{"todos":[{"title":"Todo 1","due_date":"2026-01-25"}]}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "create_todos_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			todoCreator := usecases.NewMockTodoCreator(t)
			tt.setupMocks(uow, timeProvider, todoCreator)

			action := NewBulkTodoCreatorAction(uow, todoCreator, timeProvider)
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
