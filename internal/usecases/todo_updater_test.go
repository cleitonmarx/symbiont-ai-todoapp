package usecases

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoUpdaterImpl_Update(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		ID:        fixedUUID,
		Title:     "Updated Todo",
		Status:    domain.TodoStatus_OPEN,
		Embedding: []float64{0.4, 0.5, 0.6},
		DueDate:   fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(
			uow *domain.MockUnitOfWork,
			timeProvider *domain.MockCurrentTimeProvider,
			llmClient *domain.MockLLMClient)
		id           uuid.UUID
		title        *string
		status       *domain.TodoStatus
		dueDate      *time.Time
		expectedTodo domain.Todo
		expectedErr  error
	}{
		"success": {
			id:      fixedUUID,
			title:   &todo.Title,
			status:  &todo.Status,
			dueDate: &todo.DueDate,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				llmClient.EXPECT().Embed(
					mock.Anything,
					"model-name",
					"ID: 123e4567-e89b-12d3-a456-426614174000 | Title: Updated Todo | Due Date: 2024-01-01 | Status: OPEN",
				).Return(domain.EmbedResponse{Embedding: []float64{0.4, 0.5, 0.6}}, nil)

				repo := domain.NewMockTodoRepository(t)
				outbox := domain.NewMockOutboxRepository(t)

				uow.EXPECT().Todo().Return(repo)
				uow.EXPECT().Outbox().Return(outbox)

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
				repo.EXPECT().UpdateTodo(mock.Anything, mock.MatchedBy(func(t domain.Todo) bool {
					return t.ID == fixedUUID && t.Title == todo.Title && t.UpdatedAt.Equal(fixedTime)
				})).Return(nil)

				outbox.EXPECT().CreateEvent(
					mock.Anything,
					domain.TodoEvent{
						Type:   domain.TodoEventType_TODO_UPDATED,
						TodoID: fixedUUID,
					},
				).Return(nil)
			},
			expectedTodo: todo,
			expectedErr:  nil,
		},
		"invalid-update-data": {
			id:    fixedUUID,
			title: common.Ptr(""),
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				repo := domain.NewMockTodoRepository(t)

				uow.EXPECT().Todo().Return(repo)

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
			},
			expectedTodo: domain.Todo{},
			expectedErr:  domain.NewValidationErr("title cannot be empty"),
		},
		"embedding-fails": {
			id:    fixedUUID,
			title: &todo.Title,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				llmClient.EXPECT().Embed(
					mock.Anything,
					"model-name",
					"ID: 123e4567-e89b-12d3-a456-426614174000 | Title: Updated Todo | Due Date: 2024-01-01 | Status: OPEN",
				).Return(domain.EmbedResponse{}, errors.New("embedding service error"))

				repo := domain.NewMockTodoRepository(t)

				uow.EXPECT().Todo().Return(repo)

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("embedding service error"),
		},
		"todo-not-found": {
			id: fixedUUID,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				repo := domain.NewMockTodoRepository(t)
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(domain.Todo{}, false, nil)

				uow.EXPECT().Todo().Return(repo)
			},
			expectedTodo: domain.Todo{},
			expectedErr:  domain.NewNotFoundErr(fmt.Sprintf("todo with ID %s not found", fixedUUID)),
		},
		"get-todo-fails": {
			id: fixedUUID,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				repo := domain.NewMockTodoRepository(t)

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(domain.Todo{}, false, errors.New("database error"))

				uow.EXPECT().Todo().Return(repo)

			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("database error"),
		},
		"update-fails": {
			id: fixedUUID,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				llmClient.EXPECT().Embed(
					mock.Anything,
					"model-name",
					"ID: 123e4567-e89b-12d3-a456-426614174000 | Title: Updated Todo | Due Date: 2024-01-01 | Status: OPEN",
				).Return(domain.EmbedResponse{Embedding: []float64{0.4, 0.5, 0.6}}, nil)

				repo := domain.NewMockTodoRepository(t)
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
				repo.EXPECT().UpdateTodo(mock.Anything, mock.Anything).Return(errors.New("database error"))

				uow.EXPECT().Todo().Return(repo)

			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			llmClient := domain.NewMockLLMClient(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, timeProvider, llmClient)
			}

			uti := NewTodoUpdaterImpl(uow, timeProvider, llmClient, "model-name")

			got, gotErr := uti.Update(t.Context(), uow, tt.id, tt.title, tt.status, tt.dueDate)
			assert.Equal(t, tt.expectedErr, gotErr)
			if tt.expectedErr == nil {
				assert.Equal(t, tt.id, got.ID)
			}
		})
	}
}

func TestInitTodoUpdater_Initialize(t *testing.T) {
	iut := InitTodoUpdater{}

	ctx, err := iut.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredTodoUpdater, err := depend.Resolve[TodoUpdater]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredTodoUpdater)
}
