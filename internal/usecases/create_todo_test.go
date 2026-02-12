package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateTodoImpl_Execute(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	title := "New Todo"
	dueDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	expectedTodo := domain.Todo{
		ID:        fixedUUID,
		Title:     title,
		Status:    domain.TodoStatus_OPEN,
		DueDate:   dueDate,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	tests := map[string]struct {
		title           string
		dueDate         time.Time
		setExpectations func(
			uow *domain.MockUnitOfWork,
			creator *MockTodoCreator)
		expectedTodo domain.Todo
		expectedErr  error
	}{
		"success": {
			title:   title,
			dueDate: dueDate,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				creator *MockTodoCreator,
			) {
				creator.EXPECT().
					Create(mock.Anything, uow, title, dueDate).
					Return(expectedTodo, nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})
			},
			expectedTodo: expectedTodo,
			expectedErr:  nil,
		},
		"creator-error": {
			title:   title,
			dueDate: dueDate,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				creator *MockTodoCreator,
			) {
				creator.EXPECT().
					Create(mock.Anything, uow, title, dueDate).
					Return(domain.Todo{}, errors.New("creation failed"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("creation failed"),
		},
		"uow-execute-error": {
			title:   title,
			dueDate: dueDate,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				creator *MockTodoCreator,
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
			uow := domain.NewMockUnitOfWork(t)
			creator := NewMockTodoCreator(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, creator)
			}

			cti := NewCreateTodoImpl(uow, creator)

			got, gotErr := cti.Execute(context.Background(), tt.title, tt.dueDate)
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

func TestInitCreateTodo_Initialize(t *testing.T) {
	// Clean up previous registrations if any
	ict := InitCreateTodo{}

	ctx, err := ict.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredCreateTodo, err := depend.Resolve[CreateTodo]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredCreateTodo)
}
