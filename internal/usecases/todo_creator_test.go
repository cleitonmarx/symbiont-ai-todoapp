package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoCreatorImpl_Create(t *testing.T) {
	fixedUUID := func() uuid.UUID {
		return uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		ID:        fixedUUID(),
		Title:     "My new todo",
		Status:    domain.TodoStatus_OPEN,
		Embedding: []float64{0.1, 0.2, 0.3},
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
		DueDate:   fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(
			uow *domain.MockUnitOfWork,
			timeProvider *domain.MockCurrentTimeProvider,
			llmClient *domain.MockLLMClient,
		)
		title        string
		dueDate      time.Time
		expectedTodo domain.Todo
		expectedErr  error
	}{
		"success": {
			title:   "My new todo",
			dueDate: fixedTime,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				repo := domain.NewMockTodoRepository(t)
				outbox := domain.NewMockOutboxRepository(t)

				llmClient.EXPECT().Embed(
					mock.Anything,
					"model-name",
					"ID: 123e4567-e89b-12d3-a456-426614174000 | Title: My new todo | Due Date: 2024-01-01 | Status: OPEN",
				).Return([]float64{0.1, 0.2, 0.3}, nil)

				uow.EXPECT().Todo().Return(repo)
				uow.EXPECT().Outbox().Return(outbox)

				repo.EXPECT().CreateTodo(
					mock.Anything,
					todo,
				).Return(nil)

				outbox.EXPECT().RecordEvent(
					mock.Anything,
					domain.TodoEvent{
						Type:      domain.TodoEventType_TODO_CREATED,
						TodoID:    fixedUUID(),
						CreatedAt: fixedTime,
					},
				).Return(nil)
			},
			expectedTodo: todo,
			expectedErr:  nil,
		},
		"validation-error-short-title": {
			title:   "Hi",
			dueDate: fixedTime,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)
			},
			expectedTodo: domain.Todo{},
			expectedErr:  domain.NewValidationErr("title must be between 3 and 200 characters"),
		},
		"embedding-error": {
			title:   "My new todo",
			dueDate: fixedTime,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				llmClient.EXPECT().Embed(
					mock.Anything,
					"model-name",
					"ID: 123e4567-e89b-12d3-a456-426614174000 | Title: My new todo | Due Date: 2024-01-01 | Status: OPEN",
				).Return(nil, errors.New("LLM service unavailable"))
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("LLM service unavailable"),
		},
		"repository-error": {
			title:   "My new todo",
			dueDate: fixedTime,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				timeProvider *domain.MockCurrentTimeProvider,
				llmClient *domain.MockLLMClient,
			) {

				timeProvider.EXPECT().Now().Return(fixedTime)

				repo := domain.NewMockTodoRepository(t)
				llmClient.EXPECT().Embed(
					mock.Anything,
					"model-name",
					"ID: 123e4567-e89b-12d3-a456-426614174000 | Title: My new todo | Due Date: 2024-01-01 | Status: OPEN",
				).Return([]float64{0.1, 0.2, 0.3}, nil)

				uow.EXPECT().Todo().Return(repo)

				repo.EXPECT().CreateTodo(
					mock.Anything,
					todo,
				).Return(errors.New("database error"))
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

			cti := NewTodoCreatorImpl(uow, timeProvider, llmClient, "model-name")
			cti.createUUID = fixedUUID

			got, gotErr := cti.Create(context.Background(), uow, tt.title, tt.dueDate)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.Equal(t, tt.expectedTodo, got)
		})
	}
}

func TestInitTodoCreator_Initialize(t *testing.T) {
	ict := InitTodoCreator{}

	ctx, err := ict.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredTodoCreator, err := depend.Resolve[TodoCreator]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredTodoCreator)

}
