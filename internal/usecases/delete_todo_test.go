package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteTodo_Execute(t *testing.T) {
	ctx := context.Background()
	todoID := uuid.New()
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setupMocks func(*mocks.MockUnitOfWork, *mocks.MockTodoRepository, *mocks.MockOutboxRepository)
		expectErr  bool
		validateFn func(*testing.T, *mocks.MockUnitOfWork, *mocks.MockTodoRepository, *mocks.MockOutboxRepository)
	}{
		"success-delete-todo": {
			setupMocks: func(uow *mocks.MockUnitOfWork, todoRepo *mocks.MockTodoRepository, outboxRepo *mocks.MockOutboxRepository) {
				uow.EXPECT().Execute(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, fn func(domain.UnitOfWork) error) {
						fn(uow) //nolint:errcheck
					}).
					Return(nil)

				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{ID: todoID, Title: "Test Todo"}, nil)

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
			validateFn: func(t *testing.T, uow *mocks.MockUnitOfWork, todoRepo *mocks.MockTodoRepository, outboxRepo *mocks.MockOutboxRepository) {
				uow.AssertExpectations(t)
				todoRepo.AssertExpectations(t)
				outboxRepo.AssertExpectations(t)
			},
		},
		"error-getting-todo": {
			setupMocks: func(uow *mocks.MockUnitOfWork, todoRepo *mocks.MockTodoRepository, outboxRepo *mocks.MockOutboxRepository) {
				uow.EXPECT().Execute(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, fn func(domain.UnitOfWork) error) {
						fn(uow) //nolint:errcheck
					}).
					Return(assert.AnError)

				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{}, assert.AnError)
			},
			expectErr: true,
		},
		"error-delete-todo-fails": {
			setupMocks: func(uow *mocks.MockUnitOfWork, todoRepo *mocks.MockTodoRepository, outboxRepo *mocks.MockOutboxRepository) {
				uow.EXPECT().Execute(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, fn func(domain.UnitOfWork) error) {
						fn(uow) //nolint:errcheck
					}).
					Return(assert.AnError)

				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{ID: todoID}, nil)

				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().DeleteTodo(mock.Anything, todoID).
					Return(assert.AnError)
			},
			expectErr: true,
		},
		"error-record-event-fails": {
			setupMocks: func(uow *mocks.MockUnitOfWork, todoRepo *mocks.MockTodoRepository, outboxRepo *mocks.MockOutboxRepository) {
				uow.EXPECT().Execute(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, fn func(domain.UnitOfWork) error) {
						fn(uow) //nolint:errcheck
					}).
					Return(assert.AnError)

				uow.EXPECT().Todo().Return(todoRepo)
				todoRepo.EXPECT().GetTodo(mock.Anything, todoID).
					Return(domain.Todo{ID: todoID}, nil)

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
			uow := mocks.NewMockUnitOfWork(t)
			todoRepo := mocks.NewMockTodoRepository(t)
			outboxRepo := mocks.NewMockOutboxRepository(t)
			timeProvider := mocks.NewMockCurrentTimeProvider(t)

			timeProvider.EXPECT().Now().Return(fixedTime).Maybe()

			tt.setupMocks(uow, todoRepo, outboxRepo)

			useCase := NewDeleteTodo(uow, timeProvider)
			err := useCase.Execute(ctx, todoID)

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

func TestInitDeleteTodo_Initialize(t *testing.T) {
	i := InitDeleteTodo{
		Uow:          mocks.NewMockUnitOfWork(t),
		TimeProvider: mocks.NewMockCurrentTimeProvider(t),
	}

	ctx, err := i.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify that the DeleteTodo use case is registered
	deleteTodoUseCase, err := depend.Resolve[DeleteTodo]()
	assert.NoError(t, err)
	assert.NotNil(t, deleteTodoUseCase)
}
