package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoCreatorTool(t *testing.T) {
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setupMocks func(
			*domain.MockUnitOfWork,
			*domain.MockCurrentTimeProvider,
			*MockTodoCreator,
		)
		functionCall domain.LLMStreamEventToolCall
		history      []domain.LLMChatMessage
		validateResp func(t *testing.T, resp domain.LLMChatMessage)
	}{
		"create-todo-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
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
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": "2026-01-25"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "created successfully")
			},
		},
		"create-todo-empty-due-date-uses-history": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
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
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": ""}`,
			},
			history: []domain.LLMChatMessage{
				{Role: domain.ChatRole_User, Content: "Please set it for tomorrow"},
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "created successfully")
			},
		},
		"create-todo-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
			},
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "create_todo",
				Arguments: `invalid json`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"create-todo-invalid-due-date": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": "invalid"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_date")
			},
		},
		"create-todo-create-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
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
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": "2026-01-25"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "create_todo_error")
			},
		},
		"create-todo-uow-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					Return(errors.New("uow error")).
					Once()
			},
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": "2026-01-25"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "create_todo_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			todoCreator := NewMockTodoCreator(t)
			tt.setupMocks(uow, timeProvider, todoCreator)

			tool := NewTodoCreatorTool(uow, todoCreator, timeProvider)

			resp := tool.Call(context.Background(), tt.functionCall, tt.history)
			tt.validateResp(t, resp)
		})
	}
}
