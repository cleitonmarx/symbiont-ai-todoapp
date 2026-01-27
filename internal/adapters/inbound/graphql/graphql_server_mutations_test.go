package graphql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/types"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	testNow    = time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	testID     = uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	testTitle  = "Buy groceries"
	testStatus = domain.TodoStatus_DONE
	testTodo   = domain.Todo{
		ID:        testID,
		Title:     testTitle,
		Status:    testStatus,
		DueDate:   testNow,
		CreatedAt: testNow,
		UpdatedAt: testNow,
	}
	testGenTodo = gen.Todo{
		ID:        testID,
		Title:     testTitle,
		Status:    gen.TodoStatus(testStatus),
		DueDate:   types.Date(testNow),
		CreatedAt: testNow,
		UpdatedAt: testNow,
	}
)

func TestTodoGraphQLServer_UpdateTodo(t *testing.T) {
	tests := map[string]struct {
		params      gen.UpdateTodoParams
		setupMocks  func(*mocks.MockUpdateTodo)
		expected    *gen.Todo
		expectError bool
	}{
		"success": {
			params: gen.UpdateTodoParams{
				ID:      testID,
				Title:   &testTitle,
				Status:  (*gen.TodoStatus)(&testStatus),
				DueDate: (*types.Date)(&testNow),
			},
			setupMocks: func(m *mocks.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, testID, &testTitle, (*domain.TodoStatus)(&testStatus), (*time.Time)(&testNow)).
					Return(testTodo, nil)
			},
			expected:    &testGenTodo,
			expectError: false,
		},
		"error": {
			params: gen.UpdateTodoParams{
				ID: testID,
			},
			setupMocks: func(m *mocks.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, testID, (*string)(nil), (*domain.TodoStatus)(nil), (*time.Time)(nil)).
					Return(domain.Todo{}, errors.New("fail"))
			},
			expected:    nil,
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUC := mocks.NewMockUpdateTodo(t)
			tt.setupMocks(mockUC)
			server := &TodoGraphQLServer{UpdateTodoUsecase: mockUC}

			got, err := server.UpdateTodo(context.Background(), tt.params)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestTodoGraphQLServer_DeleteTodo(t *testing.T) {
	tests := map[string]struct {
		setupMocks  func(*mocks.MockDeleteTodo)
		expect      bool
		expectError bool
	}{
		"success": {
			setupMocks: func(m *mocks.MockDeleteTodo) {
				m.EXPECT().Execute(mock.Anything, testID).Return(nil)
			},
			expect:      true,
			expectError: false,
		},
		"error": {
			setupMocks: func(m *mocks.MockDeleteTodo) {
				m.EXPECT().Execute(mock.Anything, testID).Return(errors.New("fail"))
			},
			expect:      false,
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUC := mocks.NewMockDeleteTodo(t)
			tt.setupMocks(mockUC)
			server := &TodoGraphQLServer{DeleteTodoUsecase: mockUC}

			got, err := server.DeleteTodo(context.Background(), testID)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expect, got)
			}
			mockUC.AssertExpectations(t)
		})
	}
}
