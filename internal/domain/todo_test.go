package domain

import (
	"testing"
	"time"

	"strings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTodo_ToLLMInput(t *testing.T) {
	todo := Todo{
		ID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Title:   "Finish the report",
		Status:  TodoStatus_OPEN,
		DueDate: time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC),
	}

	result := todo.ToLLMInput()
	assert.Equal(t, "ID: 00000000-0000-0000-0000-000000000001 | Title: Finish the report | Due Date: 2024-07-15 | Status: OPEN", result)
}

func TestTodo_Validate(t *testing.T) {
	now := time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		todo    Todo
		now     time.Time
		wantErr bool
		errMsg  string
	}{
		"valid-todo-open": {
			todo:    Todo{Title: "Finish report", Status: TodoStatus_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: false,
		},
		"valid-todo-done": {
			todo:    Todo{Title: "Finish report", Status: TodoStatus_DONE, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: false,
		},
		"empty-title": {
			todo:    Todo{Title: "", Status: TodoStatus_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "title cannot be empty",
		},
		"title-too-short": {
			todo:    Todo{Title: "Hi", Status: TodoStatus_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "title must be between 3 and 200 characters",
		},
		"title-too-long": {
			todo:    Todo{Title: strings.Repeat("a", 201), Status: TodoStatus_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "title must be between 3 and 200 characters",
		},
		"empty-due-date": {
			todo:    Todo{Title: "Finish report", Status: TodoStatus_OPEN, DueDate: time.Time{}},
			now:     now,
			wantErr: true,
			errMsg:  "due_date cannot be empty",
		},
		"due-date-more-than-48h-in-the-past": {
			todo:    Todo{Title: "Finish report", Status: TodoStatus_OPEN, DueDate: now.Add(-49 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "due_date cannot be more than 48 hours in the past",
		},
		"invalid-status": {
			todo:    Todo{Title: "Finish report", Status: "IN_PROGRESS", DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "status must be either OPEN or DONE",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.todo.Validate(tt.now)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListTodoOptions_WithStatus_WithEmbedding(t *testing.T) {
	status := TodoStatus_OPEN
	embedding := []float64{0.1, 0.2, 0.3}

	tests := map[string]struct {
		opts     []ListTodoOptions
		wantStat *TodoStatus
		wantEmb  []float64
	}{
		"with-status-and-embedding": {
			opts:     []ListTodoOptions{WithStatus(status), WithEmbedding(embedding)},
			wantStat: &status,
			wantEmb:  embedding,
		},
		"with-status-only": {
			opts:     []ListTodoOptions{WithStatus(status)},
			wantStat: &status,
			wantEmb:  nil,
		},
		"with-embedding-only": {
			opts:     []ListTodoOptions{WithEmbedding(embedding)},
			wantStat: nil,
			wantEmb:  embedding,
		},
		"with-no-options": {
			opts:     nil,
			wantStat: nil,
			wantEmb:  nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			params := &ListTodosParams{}
			for _, opt := range tt.opts {
				opt(params)
			}
			if tt.wantStat != nil {
				if assert.NotNil(t, params.Status, "Status should not be nil") {
					assert.Equal(t, *tt.wantStat, *params.Status)
				}
			} else {
				assert.Nil(t, params.Status)
			}
			assert.Equal(t, tt.wantEmb, params.Embedding)
		})
	}
}
