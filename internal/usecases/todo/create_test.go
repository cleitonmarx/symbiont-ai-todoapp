package todo

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateImpl_Execute(t *testing.T) {
	t.Parallel()

	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	title := "New Todo"
	dueDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	expectedTodo := domain.Todo{
		ID:        fixedUUID,
		Title:     title,
		Status:    domain.Status_OPEN,
		DueDate:   dueDate,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	tests := map[string]struct {
		title           string
		dueDate         time.Time
		setExpectations func(
			uow *transaction.MockUnitOfWork,
			creator *MockCreator)
		expectedTodo domain.Todo
		expectedErr  error
	}{
		"success": {
			title:   title,
			dueDate: dueDate,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				creator *MockCreator,
			) {
				creator.EXPECT().
					Create(mock.Anything, mock.Anything, title, dueDate).
					Return(expectedTodo, nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uowCtx context.Context, scope transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedTodo: expectedTodo,
			expectedErr:  nil,
		},
		"creator-error": {
			title:   title,
			dueDate: dueDate,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				creator *MockCreator,
			) {
				creator.EXPECT().
					Create(mock.Anything, mock.Anything, title, dueDate).
					Return(domain.Todo{}, errors.New("creation failed"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("creation failed"),
		},
		"uow-execute-error": {
			title:   title,
			dueDate: dueDate,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				creator *MockCreator,
			) {
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					Return(errors.New("transaction failed"))
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("transaction failed"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := transaction.NewMockUnitOfWork(t)
			creator := NewMockCreator(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, creator)
			}

			cti := NewCreateImpl(uow, creator)

			got, gotErr := cti.Execute(t.Context(), tt.title, tt.dueDate)
			assert.Equal(t, tt.expectedErr, gotErr)
			if tt.expectedErr == nil {
				assert.Equal(t, tt.expectedTodo.ID, got.ID)
				assert.Equal(t, tt.expectedTodo.Title, got.Title)
				assert.Equal(t, tt.expectedTodo.Status, got.Status)
				assert.Equal(t, tt.expectedTodo.DueDate, got.DueDate)
			}
		})
	}
}
