package todo

import (
	"context"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleterImpl_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	todoID := uuid.New()
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setupDomain func(*transaction.MockScope, *domain.MockRepository, *outbox.MockRepository)
		expectErr   bool
	}{
		"success-delete-todo": {
			setupDomain: func(scope *transaction.MockScope, todoRepo *domain.MockRepository, outboxRepo *outbox.MockRepository) {

				scope.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{ID: todoID, Title: "Test Todo"}, true, nil)

				scope.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().DeleteTodo(mock.Anything, todoID).Return(nil)

				scope.EXPECT().Outbox().Return(outboxRepo)
				outboxRepo.EXPECT().CreateTodoEvent(mock.Anything, mock.MatchedBy(func(event outbox.TodoEvent) bool {
					return event.Type == outbox.EventType_TODO_DELETED &&
						event.TodoID == todoID &&
						event.CreatedAt.Equal(fixedTime)
				})).Return(nil)
			},
			expectErr: false,
		},
		"todo-not-found": {
			setupDomain: func(scope *transaction.MockScope, todoRepo *domain.MockRepository, outboxRepo *outbox.MockRepository) {
				scope.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{}, false, nil)
			},
			expectErr: true,
		},
		"error-getting-todo": {
			setupDomain: func(scope *transaction.MockScope, todoRepo *domain.MockRepository, outboxRepo *outbox.MockRepository) {
				scope.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{}, false, assert.AnError)
			},
			expectErr: true,
		},
		"error-delete-todo-fails": {
			setupDomain: func(scope *transaction.MockScope, todoRepo *domain.MockRepository, outboxRepo *outbox.MockRepository) {
				scope.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{ID: todoID}, true, nil)

				scope.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().DeleteTodo(mock.Anything, todoID).
					Return(assert.AnError)
			},
			expectErr: true,
		},
		"error-record-event-fails": {
			setupDomain: func(scope *transaction.MockScope, todoRepo *domain.MockRepository, outboxRepo *outbox.MockRepository) {
				scope.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{ID: todoID}, true, nil)

				scope.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().DeleteTodo(mock.Anything, todoID).Return(nil)

				scope.EXPECT().Outbox().Return(outboxRepo)
				outboxRepo.EXPECT().CreateTodoEvent(mock.Anything, mock.Anything).
					Return(assert.AnError)
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			scope := transaction.NewMockScope(t)
			todoRepo := domain.NewMockRepository(t)
			outboxRepo := outbox.NewMockRepository(t)
			timeProvider := core.NewMockCurrentTimeProvider(t)

			timeProvider.EXPECT().Now().Return(fixedTime).Maybe()

			tt.setupDomain(scope, todoRepo, outboxRepo)

			deleter := NewDeleterImpl(timeProvider)
			err := deleter.Delete(ctx, scope, todoID)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
