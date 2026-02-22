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

func TestBulkTodoDueDateUpdaterAction(t *testing.T) {
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)
	todoID1 := uuid.New()
	todoID2 := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*domain.MockUnitOfWork, *domain.MockCurrentTimeProvider, *usecases.MockTodoUpdater)
		functionCall domain.AssistantActionCall
		history      []domain.AssistantMessage
		validateResp func(t *testing.T, resp domain.AssistantMessage)
	}{
		"update-todos-due-date-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *usecases.MockTodoUpdater) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID1,
						(*string)(nil),
						(*domain.TodoStatus)(nil),
						common.Ptr(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
					).
					Return(
						domain.Todo{
							ID:      todoID1,
							Title:   "Todo 1",
							DueDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
							Status:  domain.TodoStatus_OPEN,
						},
						nil,
					).
					Once()
				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID2,
						(*string)(nil),
						(*domain.TodoStatus)(nil),
						common.Ptr(time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)),
					).
					Return(
						domain.Todo{
							ID:      todoID2,
							Title:   "Todo 2",
							DueDate: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
							Status:  domain.TodoStatus_OPEN,
						},
						nil,
					).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "update_todos_due_date",
				Input: `{"todos":[{"id":"` + todoID1.String() + `","due_date":"2026-02-01"},{"id":"` + todoID2.String() + `","due_date":"2026-02-02"}]}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "todos[2]{id,title,due_date,status}")
			},
		},
		"update-todos-due-date-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *usecases.MockTodoUpdater) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "update_todos_due_date",
				Input: `invalid json`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"update-todos-due-date-invalid-due-date": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *usecases.MockTodoUpdater) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "update_todos_due_date",
				Input: `{"todos":[{"id":"` + todoID1.String() + `","due_date":"invalid"}]}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "invalid_due_date")
			},
		},
		"update-todos-due-date-update-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *usecases.MockTodoUpdater) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID1,
						(*string)(nil),
						(*domain.TodoStatus)(nil),
						common.Ptr(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
					).
					Return(domain.Todo{}, errors.New("update error")).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "update_todos_due_date",
				Input: `{"todos":[{"id":"` + todoID1.String() + `","due_date":"2026-02-01"}]}`,
			},
			history: []domain.AssistantMessage{},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Contains(t, resp.Content, "update_todos_due_date_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			updater := usecases.NewMockTodoUpdater(t)
			tt.setupMocks(uow, timeProvider, updater)

			action := NewBulkTodoDueDateUpdaterAction(uow, updater, timeProvider)
			assert.NotEmpty(t, action.StatusMessage())

			definition := action.Definition()
			assert.Equal(t, "update_todos_due_date", definition.Name)
			assert.NotEmpty(t, definition.Description)
			assert.NotEmpty(t, definition.Input)

			resp := action.Execute(context.Background(), tt.functionCall, tt.history)
			tt.validateResp(t, resp)
		})
	}
}
