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

func TestUpdateImpl_Execute(t *testing.T) {
	t.Parallel()

	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	newTitle := "Updated Todo"
	newStatus := domain.Status_DONE
	newDueDate := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)

	expectedTodo := domain.Todo{
		ID:        fixedUUID,
		Title:     newTitle,
		Status:    newStatus,
		DueDate:   newDueDate,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	tests := map[string]struct {
		id              uuid.UUID
		title           *string
		status          *domain.Status
		dueDate         *time.Time
		setExpectations func(
			uow *transaction.MockUnitOfWork,
			modifier *MockUpdater,
		)
		expectedTodo domain.Todo
		expectedErr  error
	}{
		"success-update-all-fields": {
			id:      fixedUUID,
			title:   &newTitle,
			status:  &newStatus,
			dueDate: &newDueDate,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				modifier *MockUpdater,
			) {
				modifier.EXPECT().
					Update(mock.Anything, mock.Anything, fixedUUID, &newTitle, &newStatus, &newDueDate).
					Return(expectedTodo, nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedTodo: expectedTodo,
			expectedErr:  nil,
		},
		"success-update-title-only": {
			id:      fixedUUID,
			title:   &newTitle,
			status:  nil,
			dueDate: nil,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				modifier *MockUpdater,
			) {

				modifier.EXPECT().
					Update(mock.Anything, mock.Anything, fixedUUID, &newTitle, (*domain.Status)(nil), (*time.Time)(nil)).
					Return(expectedTodo, nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedTodo: expectedTodo,
			expectedErr:  nil,
		},
		"success-update-status-only": {
			id:      fixedUUID,
			title:   nil,
			status:  &newStatus,
			dueDate: nil,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				modifier *MockUpdater,
			) {
				modifier.EXPECT().
					Update(mock.Anything, mock.Anything, fixedUUID, (*string)(nil), &newStatus, (*time.Time)(nil)).
					Return(expectedTodo, nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedTodo: expectedTodo,
			expectedErr:  nil,
		},
		"success-update-duedate-only": {
			id:      fixedUUID,
			title:   nil,
			status:  nil,
			dueDate: &newDueDate,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				modifier *MockUpdater,
			) {
				modifier.EXPECT().
					Update(mock.Anything, mock.Anything, fixedUUID, (*string)(nil), (*domain.Status)(nil), &newDueDate).
					Return(expectedTodo, nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedTodo: expectedTodo,
			expectedErr:  nil,
		},
		"modifier-error-not-found": {
			id:      fixedUUID,
			title:   &newTitle,
			status:  nil,
			dueDate: nil,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				modifier *MockUpdater,
			) {
				modifier.EXPECT().
					Update(mock.Anything, mock.Anything, fixedUUID, &newTitle, (*domain.Status)(nil), (*time.Time)(nil)).
					Return(domain.Todo{}, errors.New("todo not found"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("todo not found"),
		},
		"modifier-error-validation": {
			id:      fixedUUID,
			title:   &newTitle,
			status:  &newStatus,
			dueDate: &newDueDate,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				modifier *MockUpdater,
			) {
				modifier.EXPECT().
					Update(mock.Anything, mock.Anything, fixedUUID, &newTitle, &newStatus, &newDueDate).
					Return(domain.Todo{}, errors.New("validation failed"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, transaction.NewMockScope(t))
					})
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("validation failed"),
		},
		"uow-execute-error": {
			id:      fixedUUID,
			title:   &newTitle,
			status:  nil,
			dueDate: nil,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				modifier *MockUpdater,
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
			modifier := NewMockUpdater(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, modifier)
			}

			uti := NewUpdateImpl(uow, modifier)

			got, gotErr := uti.Execute(t.Context(), tt.id, tt.title, tt.status, tt.dueDate)
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
