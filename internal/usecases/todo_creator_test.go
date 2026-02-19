package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
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
			semanticEncoder *domain.MockSemanticEncoder,
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
				semanticEncoder *domain.MockSemanticEncoder,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				repo := domain.NewMockTodoRepository(t)
				outbox := domain.NewMockOutboxRepository(t)

				semanticEncoder.EXPECT().VectorizeTodo(
					mock.Anything,
					"model-name",
					mock.MatchedBy(func(t domain.Todo) bool {
						return t.Title == todo.Title && t.DueDate.Equal(todo.DueDate)
					}),
				).Return(domain.EmbeddingVector{Vector: []float64{0.1, 0.2, 0.3}}, nil)

				uow.EXPECT().Todo().Return(repo)
				uow.EXPECT().Outbox().Return(outbox)

				repo.EXPECT().CreateTodo(
					mock.Anything,
					mock.MatchedBy(func(t domain.Todo) bool {
						return t.Title == todo.Title && t.DueDate.Equal(todo.DueDate)
					}),
				).Return(nil)

				outbox.EXPECT().CreateTodoEvent(
					mock.Anything,
					domain.TodoEvent{
						Type:      domain.EventType_TODO_CREATED,
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
				semanticEncoder *domain.MockSemanticEncoder,
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
				semanticEncoder *domain.MockSemanticEncoder,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				semanticEncoder.EXPECT().VectorizeTodo(
					mock.Anything,
					"model-name",
					mock.MatchedBy(func(t domain.Todo) bool {
						return t.Title == todo.Title && t.DueDate.Equal(todo.DueDate)
					}),
				).Return(domain.EmbeddingVector{}, errors.New("LLM service unavailable"))
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
				semanticEncoder *domain.MockSemanticEncoder,
			) {

				timeProvider.EXPECT().Now().Return(fixedTime)

				repo := domain.NewMockTodoRepository(t)
				semanticEncoder.EXPECT().VectorizeTodo(
					mock.Anything,
					"model-name",
					mock.MatchedBy(func(t domain.Todo) bool {
						return t.Title == todo.Title && t.DueDate.Equal(todo.DueDate)
					}),
				).Return(domain.EmbeddingVector{Vector: []float64{0.1, 0.2, 0.3}}, nil)

				uow.EXPECT().Todo().Return(repo)

				repo.EXPECT().CreateTodo(
					mock.Anything,
					mock.MatchedBy(func(t domain.Todo) bool {
						return t.Title == todo.Title && t.DueDate.Equal(todo.DueDate)
					}),
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
			semanticEncoder := domain.NewMockSemanticEncoder(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, timeProvider, semanticEncoder)
			}

			cti := NewTodoCreatorImpl(uow, timeProvider, semanticEncoder, "model-name")
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
