package todo

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdaterImpl_Update(t *testing.T) {
	t.Parallel()

	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		ID:        fixedUUID,
		Title:     "Updated Todo",
		Status:    domain.Status_OPEN,
		Embedding: []float64{0.4, 0.5, 0.6},
		DueDate:   fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(
			scope *transaction.MockScope,
			timeProvider *core.MockCurrentTimeProvider,
			semanticEncoder *semantic.MockEncoder)
		id           uuid.UUID
		title        *string
		status       *domain.Status
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
				).Return(semantic.EmbeddingVector{Vector: []float64{0.4, 0.5, 0.6}}, nil)

				repo := domain.NewMockRepository(t)
				outboxRepo := outbox.NewMockRepository(t)

				scope.EXPECT().Todo().Return(repo)
				scope.EXPECT().Outbox().Return(outboxRepo)

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
				repo.EXPECT().UpdateTodo(mock.Anything, mock.MatchedBy(func(t domain.Todo) bool {
					return t.ID == fixedUUID && t.Title == todo.Title && t.UpdatedAt.Equal(fixedTime)
				})).Return(nil)

				outboxRepo.EXPECT().CreateTodoEvent(
					mock.Anything,
					outbox.TodoEvent{
						Type:      outbox.EventType_TODO_UPDATED,
						TodoID:    fixedUUID,
						CreatedAt: fixedTime,
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
				scope *transaction.MockScope,
				timeProvider *core.MockCurrentTimeProvider,
				semanticEncoder *semantic.MockEncoder,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				repo := domain.NewMockRepository(t)

				scope.EXPECT().Todo().Return(repo)

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
			},
			expectedTodo: domain.Todo{},
			expectedErr:  core.NewValidationErr("title cannot be empty"),
		},
		"embedding-fails": {
			id:    fixedUUID,
			title: &todo.Title,
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
				).Return(semantic.EmbeddingVector{}, errors.New("embedding service error"))

				repo := domain.NewMockRepository(t)

				scope.EXPECT().Todo().Return(repo)

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("embedding service error"),
		},
		"todo-not-found": {
			id: fixedUUID,
			setExpectations: func(
				scope *transaction.MockScope,
				timeProvider *core.MockCurrentTimeProvider,
				semanticEncoder *semantic.MockEncoder,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				repo := domain.NewMockRepository(t)
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(domain.Todo{}, false, nil)

				scope.EXPECT().Todo().Return(repo)
			},
			expectedTodo: domain.Todo{},
			expectedErr:  core.NewNotFoundErr(fmt.Sprintf("todo with ID %s not found", fixedUUID)),
		},
		"get-todo-fails": {
			id: fixedUUID,
			setExpectations: func(
				scope *transaction.MockScope,
				timeProvider *core.MockCurrentTimeProvider,
				semanticEncoder *semantic.MockEncoder,
			) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				repo := domain.NewMockRepository(t)

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(domain.Todo{}, false, errors.New("database error"))

				scope.EXPECT().Todo().Return(repo)

			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("database error"),
		},
		"update-fails": {
			id: fixedUUID,
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
				).Return(semantic.EmbeddingVector{Vector: []float64{0.4, 0.5, 0.6}}, nil)

				repo := domain.NewMockRepository(t)
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
				repo.EXPECT().UpdateTodo(mock.Anything, mock.Anything).Return(errors.New("database error"))

				scope.EXPECT().Todo().Return(repo)

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

			uti := NewUpdaterImpl(timeProvider, semanticEncoder, "model-name")

			got, gotErr := uti.Update(t.Context(), scope, tt.id, tt.title, tt.status, tt.dueDate)
			assert.Equal(t, tt.expectedErr, gotErr)
			if tt.expectedErr == nil {
				assert.Equal(t, tt.id, got.ID)
			}
		})
	}
}
