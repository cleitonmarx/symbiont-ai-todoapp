package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoDeleter_Delete(t *testing.T) {
	ctx := context.Background()
	todoID := uuid.New()
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setupDomain func(*domain.MockUnitOfWork, *domain.MockTodoRepository, *domain.MockOutboxRepository)
		expectErr   bool
		validateFn  func(*testing.T, *domain.MockUnitOfWork, *domain.MockTodoRepository, *domain.MockOutboxRepository)
	}{
		"success-delete-todo": {
			setupDomain: func(uow *domain.MockUnitOfWork, todoRepo *domain.MockTodoRepository, outboxRepo *domain.MockOutboxRepository) {

				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{ID: todoID, Title: "Test Todo"}, true, nil)

				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().DeleteTodo(mock.Anything, todoID).Return(nil)

				uow.EXPECT().Outbox().Return(outboxRepo)
				outboxRepo.EXPECT().RecordEvent(mock.Anything, mock.MatchedBy(func(event domain.TodoEvent) bool {
					return event.Type == domain.TodoEventType_TODO_DELETED &&
						event.TodoID == todoID &&
						event.CreatedAt.Equal(fixedTime)
				})).Return(nil)
			},
			expectErr: false,
			validateFn: func(t *testing.T, uow *domain.MockUnitOfWork, todoRepo *domain.MockTodoRepository, outboxRepo *domain.MockOutboxRepository) {
				uow.AssertExpectations(t)
				todoRepo.AssertExpectations(t)
				outboxRepo.AssertExpectations(t)
			},
		},
		"todo-not-found": {
			setupDomain: func(uow *domain.MockUnitOfWork, todoRepo *domain.MockTodoRepository, outboxRepo *domain.MockOutboxRepository) {
				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{}, false, nil)
			},
			expectErr: true,
		},
		"error-getting-todo": {
			setupDomain: func(uow *domain.MockUnitOfWork, todoRepo *domain.MockTodoRepository, outboxRepo *domain.MockOutboxRepository) {
				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{}, false, assert.AnError)
			},
			expectErr: true,
		},
		"error-delete-todo-fails": {
			setupDomain: func(uow *domain.MockUnitOfWork, todoRepo *domain.MockTodoRepository, outboxRepo *domain.MockOutboxRepository) {
				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{ID: todoID}, true, nil)

				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().DeleteTodo(mock.Anything, todoID).
					Return(assert.AnError)
			},
			expectErr: true,
		},
		"error-record-event-fails": {
			setupDomain: func(uow *domain.MockUnitOfWork, todoRepo *domain.MockTodoRepository, outboxRepo *domain.MockOutboxRepository) {
				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{ID: todoID}, true, nil)

				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().DeleteTodo(mock.Anything, todoID).Return(nil)

				uow.EXPECT().Outbox().Return(outboxRepo)
				outboxRepo.EXPECT().RecordEvent(mock.Anything, mock.Anything).
					Return(assert.AnError)
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			todoRepo := domain.NewMockTodoRepository(t)
			outboxRepo := domain.NewMockOutboxRepository(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)

			timeProvider.EXPECT().Now().Return(fixedTime).Maybe()

			tt.setupDomain(uow, todoRepo, outboxRepo)

			deleter := NewTodoDeleterImpl(uow, timeProvider)
			err := deleter.Delete(ctx, uow, todoID)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validateFn != nil {
				tt.validateFn(t, uow, todoRepo, outboxRepo)
			}
		})
	}
}

func TestInitTodoDeleter_Initialize(t *testing.T) {
	id := InitTodoDeleter{
		Uow:          domain.NewMockUnitOfWork(t),
		TimeProvider: domain.NewMockCurrentTimeProvider(t),
	}

	ctx, err := id.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify that the TodoDeleter is registered
	todoDeleter, err := depend.Resolve[TodoDeleter]()
	assert.NoError(t, err)
	assert.NotNil(t, todoDeleter)
}
