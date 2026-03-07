package todo

import (
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreatorImpl_Create(t *testing.T) {
	t.Parallel()

	fixedUUID := func() uuid.UUID {
		return uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		ID:        fixedUUID(),
		Title:     "My new todo",
		Status:    domain.Status_OPEN,
		Embedding: []float64{0.1, 0.2, 0.3},
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
		DueDate:   fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(
			scope *transaction.MockScope,
			timeProvider *core.MockCurrentTimeProvider,
			semanticEncoder *semantic.MockEncoder,
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
				scope *transaction.MockScope,
				timeProvider *core.MockCurrentTimeProvider,
				semanticEncoder *semantic.MockEncoder,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				repo := domain.NewMockRepository(t)
				outboxRepo := outbox.NewMockRepository(t)

				semanticEncoder.EXPECT().VectorizeTodo(
					mock.Anything,
					"model-name",
					mock.MatchedBy(func(t domain.Todo) bool {
						return t.Title == todo.Title && t.DueDate.Equal(todo.DueDate)
					}),
				).Return(semantic.EmbeddingVector{Vector: []float64{0.1, 0.2, 0.3}}, nil)

				scope.EXPECT().Todo().Return(repo).Once()
				scope.EXPECT().Outbox().Return(outboxRepo).Once()

				repo.EXPECT().CreateTodo(
					mock.Anything,
					mock.MatchedBy(func(t domain.Todo) bool {
						return t.Title == todo.Title && t.DueDate.Equal(todo.DueDate)
					}),
				).Return(nil)

				outboxRepo.EXPECT().CreateTodoEvent(
					mock.Anything,
					outbox.TodoEvent{
						Type:      outbox.EventType_TODO_CREATED,
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
				scope *transaction.MockScope,
				timeProvider *core.MockCurrentTimeProvider,
				semanticEncoder *semantic.MockEncoder,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)
			},
			expectedTodo: domain.Todo{},
			expectedErr:  core.NewValidationErr("title must be between 3 and 200 characters"),
		},
		"embedding-error": {
			title:   "My new todo",
			dueDate: fixedTime,
			setExpectations: func(
				scope *transaction.MockScope,
				timeProvider *core.MockCurrentTimeProvider,
				semanticEncoder *semantic.MockEncoder,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				semanticEncoder.EXPECT().VectorizeTodo(
					mock.Anything,
					"model-name",
					mock.MatchedBy(func(t domain.Todo) bool {
						return t.Title == todo.Title && t.DueDate.Equal(todo.DueDate)
					}),
				).Return(semantic.EmbeddingVector{}, errors.New("LLM service unavailable"))
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("LLM service unavailable"),
		},
		"repository-error": {
			title:   "My new todo",
			dueDate: fixedTime,
			setExpectations: func(
				scope *transaction.MockScope,
				timeProvider *core.MockCurrentTimeProvider,
				semanticEncoder *semantic.MockEncoder,
			) {

				timeProvider.EXPECT().Now().Return(fixedTime)

				repo := domain.NewMockRepository(t)
				semanticEncoder.EXPECT().VectorizeTodo(
					mock.Anything,
					"model-name",
					mock.MatchedBy(func(t domain.Todo) bool {
						return t.Title == todo.Title && t.DueDate.Equal(todo.DueDate)
					}),
				).Return(semantic.EmbeddingVector{Vector: []float64{0.1, 0.2, 0.3}}, nil)

				scope.EXPECT().Todo().Return(repo)

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
			scope := transaction.NewMockScope(t)
			timeProvider := core.NewMockCurrentTimeProvider(t)
			semanticEncoder := semantic.NewMockEncoder(t)
			if tt.setExpectations != nil {
				tt.setExpectations(scope, timeProvider, semanticEncoder)
			}

			cti := NewCreatorImpl(timeProvider, semanticEncoder, "model-name")
			cti.createUUID = fixedUUID

			got, gotErr := cti.Create(t.Context(), scope, tt.title, tt.dueDate)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.Equal(t, tt.expectedTodo, got)
		})
	}
}
