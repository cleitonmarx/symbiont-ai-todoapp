package graphql

import (
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql/types"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	testNow    = time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	testID     = uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	testTitle  = "Buy groceries"
	testStatus = todo.Status_DONE
	testTodo   = todo.Todo{
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
	t.Parallel()

	tests := map[string]struct {
		params        gen.UpdateTodoParams
		setupUsecases func(*todouc.MockUpdate)
		expected      *gen.Todo
		expectError   bool
	}{
		"success": {
			params: gen.UpdateTodoParams{
				ID:      testID,
				Title:   &testTitle,
				Status:  (*gen.TodoStatus)(&testStatus),
				DueDate: (*types.Date)(&testNow),
			},
			setupUsecases: func(m *todouc.MockUpdate) {
				m.EXPECT().
					Execute(mock.Anything, testID, &testTitle, (*todo.Status)(&testStatus), (*time.Time)(&testNow)).
					Return(testTodo, nil)
			},
			expected:    &testGenTodo,
			expectError: false,
		},
		"error": {
			params: gen.UpdateTodoParams{
				ID: testID,
			},
			setupUsecases: func(m *todouc.MockUpdate) {
				m.EXPECT().
					Execute(mock.Anything, testID, (*string)(nil), (*todo.Status)(nil), (*time.Time)(nil)).
					Return(todo.Todo{}, errors.New("fail"))
			},
			expected:    nil,
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUC := todouc.NewMockUpdate(t)
			tt.setupUsecases(mockUC)
			server := &TodoGraphQLServer{
				UpdateTodoUsecase: mockUC,
				Logger:            log.New(io.Discard, "", 0),
			}

			got, err := server.UpdateTodo(t.Context(), tt.params)
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
	t.Parallel()

	tests := map[string]struct {
		setupUsecases func(*todouc.MockDelete)
		expect        bool
		expectError   bool
	}{
		"success": {
			setupUsecases: func(m *todouc.MockDelete) {
				m.EXPECT().Execute(mock.Anything, testID).Return(nil)
			},
			expect:      true,
			expectError: false,
		},
		"error": {
			setupUsecases: func(m *todouc.MockDelete) {
				m.EXPECT().Execute(mock.Anything, testID).Return(errors.New("fail"))
			},
			expect:      false,
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUC := todouc.NewMockDelete(t)
			tt.setupUsecases(mockUC)
			server := &TodoGraphQLServer{
				DeleteTodoUsecase: mockUC,
				Logger:            log.New(io.Discard, "", 0),
			}

			got, err := server.DeleteTodo(t.Context(), testID)
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
