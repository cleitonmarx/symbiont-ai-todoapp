package todo

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteImpl_Execute(t *testing.T) {
	t.Parallel()

	todoID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := map[string]struct {
		id              uuid.UUID
		setExpectations func(
			uow *transaction.MockUnitOfWork,
			deleter *MockDeleter)
		expectedErr error
	}{
		"success": {
			id: todoID,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				deleter *MockDeleter,
			) {
				deleter.EXPECT().
					Delete(mock.Anything, mock.Anything, todoID).
					Return(nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedErr: nil,
		},
		"deleter-error": {
			id: todoID,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				deleter *MockDeleter,
			) {
				deleter.EXPECT().
					Delete(mock.Anything, mock.Anything, todoID).
					Return(errors.New("deletion failed"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedErr: errors.New("deletion failed"),
		},
		"uow-execute-error": {
			id: todoID,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				deleter *MockDeleter,
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
			uow := transaction.NewMockUnitOfWork(t)
			deleter := NewMockDeleter(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, deleter)
			}

			dti := NewDelete(uow, deleter)

			gotErr := dti.Execute(context.Background(), tt.id)
			assert.Equal(t, tt.expectedErr, gotErr)
		})
	}
}
