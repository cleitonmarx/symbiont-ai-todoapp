package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoDueDateUpdaterTool(t *testing.T) {
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)
	todoID := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*domain.MockUnitOfWork, *domain.MockCurrentTimeProvider, *MockTodoUpdater)
		functionCall domain.LLMStreamEventToolCall
		history      []domain.LLMChatMessage
		validateResp func(t *testing.T, resp domain.LLMChatMessage)
	}{
		"update-due-date-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						(*string)(nil),
						(*domain.TodoStatus)(nil),
						common.Ptr(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
					).
					Return(
						domain.Todo{
							ID:      todoID,
							Title:   "Some Todo",
							DueDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
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
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "update_todo_due_date",
				Arguments: `{"id": "` + todoID.String() + `", "due_date": "2026-02-01"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "updated successfully")
			},
		},
		"update-due-date-uses-history": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						(*string)(nil),
						(*domain.TodoStatus)(nil),
						mock.Anything,
					).
					Return(
						domain.Todo{
							ID:    todoID,
							Title: "Some Todo",
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
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "update_todo_due_date",
				Arguments: `{"id": "` + todoID.String() + `", "due_date": ""}`,
			},
			history: []domain.LLMChatMessage{
				{Role: domain.ChatRole_User, Content: "Please set it to tomorrow"},
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "updated successfully")
			},
		},
		"update-due-date-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
			},
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "update_todo_due_date",
				Arguments: `invalid json`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"update-due-date-invalid-id": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
			},
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "update_todo_due_date",
				Arguments: `{"id": "00000000-0000-0000-0000-000000000000", "due_date": "2026-02-01"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_todo_id")
			},
		},
		"update-due-date-update-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						(*string)(nil),
						(*domain.TodoStatus)(nil),
						mock.Anything,
					).
					Return(domain.Todo{}, errors.New("update error"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "update_todo_due_date",
				Arguments: `{"id": "` + todoID.String() + `", "due_date": "2026-02-01"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "update_due_date_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			updater := NewMockTodoUpdater(t)
			tt.setupMocks(uow, timeProvider, updater)

			tool := NewTodoDueDateUpdaterTool(uow, updater, timeProvider)

			resp := tool.Call(context.Background(), tt.functionCall, tt.history)
			tt.validateResp(t, resp)
		})
	}
}
