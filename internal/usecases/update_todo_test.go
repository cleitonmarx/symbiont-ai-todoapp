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
	domain_mocks "github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateTodoImpl_Execute(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		ID:      fixedUUID,
		Title:   "Updated Todo",
		Status:  domain.TodoStatus_OPEN,
		DueDate: fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(uow *domain_mocks.MockUnitOfWork, timeProvider *domain_mocks.MockCurrentTimeProvider)
		id              uuid.UUID
		title           *string
		status          *domain.TodoStatus
		dueDate         *time.Time
		expectedTodo    domain.Todo
		expectedErr     error
	}{
		"success": {
			id:      fixedUUID,
			title:   &todo.Title,
			status:  &todo.Status,
			dueDate: &todo.DueDate,
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, timeProvider *domain_mocks.MockCurrentTimeProvider) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				repo := domain_mocks.NewMockTodoRepository(t)
				outbox := domain_mocks.NewMockOutboxRepository(t)

				uow.EXPECT().Todo().Return(repo)
				uow.EXPECT().Outbox().Return(outbox)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
				repo.EXPECT().UpdateTodo(mock.Anything, mock.MatchedBy(func(t domain.Todo) bool {
					return t.ID == fixedUUID && t.Title == todo.Title && t.UpdatedAt.Equal(fixedTime)
				})).Return(nil)

				outbox.EXPECT().RecordEvent(
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
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, timeProvider *domain_mocks.MockCurrentTimeProvider) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				repo := domain_mocks.NewMockTodoRepository(t)

				uow.EXPECT().Todo().Return(repo)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})

				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
			},
			expectedTodo: domain.Todo{},
			expectedErr:  domain.NewValidationErr("title cannot be empty"),
		},
		"todo-not-found": {
			id: fixedUUID,
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, timeProvider *domain_mocks.MockCurrentTimeProvider) {
				timeProvider.EXPECT().Now().Return(fixedTime)

				repo := domain_mocks.NewMockTodoRepository(t)
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(domain.Todo{}, false, nil)

				uow.EXPECT().Todo().Return(repo)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})

			},
			expectedTodo: domain.Todo{},
			expectedErr:  domain.NewNotFoundErr(fmt.Sprintf("todo with ID %s not found", fixedUUID)),
		},
		"get-todo-fails": {
			id: fixedUUID,
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, timeProvider *domain_mocks.MockCurrentTimeProvider) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				repo := domain_mocks.NewMockTodoRepository(t)
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(domain.Todo{}, false, errors.New("database error"))

				uow.EXPECT().Todo().Return(repo)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})

			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("database error"),
		},
		"update-fails": {
			id: fixedUUID,
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, timeProvider *domain_mocks.MockCurrentTimeProvider) {
				timeProvider.EXPECT().Now().Return(fixedTime)
				repo := domain_mocks.NewMockTodoRepository(t)
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, true, nil)
				repo.EXPECT().UpdateTodo(mock.Anything, mock.Anything).Return(errors.New("database error"))

				uow.EXPECT().Todo().Return(repo)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})

			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain_mocks.NewMockUnitOfWork(t)
			timeProvider := domain_mocks.NewMockCurrentTimeProvider(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, timeProvider)
			}

			uti := NewUpdateTodoImpl(uow, timeProvider)

			got, gotErr := uti.Execute(context.Background(), tt.id, tt.title, tt.status, tt.dueDate)
			assert.Equal(t, tt.expectedErr, gotErr)
			if tt.expectedErr == nil {
				assert.Equal(t, tt.id, got.ID)
			}
		})
	}
}

func TestInitUpdateTodo_Initialize(t *testing.T) {
	iut := InitUpdateTodo{}

	ctx, err := iut.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredUpdateTodo, err := depend.Resolve[UpdateTodo]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredUpdateTodo)
}
