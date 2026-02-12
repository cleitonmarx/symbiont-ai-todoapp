package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteTodoImpl_Execute(t *testing.T) {
	todoID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := map[string]struct {
		id              uuid.UUID
		setExpectations func(
			uow *domain.MockUnitOfWork,
			deleter *MockTodoDeleter)
		expectedErr error
	}{
		"success": {
			id: todoID,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				deleter *MockTodoDeleter,
			) {
				deleter.EXPECT().
					Delete(mock.Anything, uow, todoID).
					Return(nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})
			},
			expectedErr: nil,
		},
		"deleter-error": {
			id: todoID,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				deleter *MockTodoDeleter,
			) {
				deleter.EXPECT().
					Delete(mock.Anything, uow, todoID).
					Return(errors.New("deletion failed"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})
			},
			expectedErr: errors.New("deletion failed"),
		},
		"uow-execute-error": {
			id: todoID,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				deleter *MockTodoDeleter,
			) {
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					Return(errors.New("transaction failed"))
			},
			expectedErr: errors.New("transaction failed"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			deleter := NewMockTodoDeleter(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, deleter)
			}

			dti := NewDeleteTodo(uow, deleter)

			gotErr := dti.Execute(context.Background(), tt.id)
			assert.Equal(t, tt.expectedErr, gotErr)
		})
	}
}

func TestInitDeleteTodo_Initialize(t *testing.T) {
	// Clean up previous registrations if any
	idt := InitDeleteTodo{}

	ctx, err := idt.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredDeleteTodo, err := depend.Resolve[DeleteTodo]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredDeleteTodo)
}
