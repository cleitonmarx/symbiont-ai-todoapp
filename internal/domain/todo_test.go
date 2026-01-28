package domain

import (
	"testing"
	"time"

	"strings"

	"github.com/stretchr/testify/assert"
)

func TestTodo_ToLLMInput(t *testing.T) {
	todo := Todo{
		Title:   "Finish the report",
		Status:  TodoStatus_OPEN,
		DueDate: time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC),
	}

	result := todo.ToLLMInput()
	assert.Equal(t, "Task: Finish the report | Status: OPEN | Due: 2024-07-15", result)
}

func TestTodo_Validate(t *testing.T) {
	now := time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		todo    Todo
		now     time.Time
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid-todo-open",
			todo:    Todo{Title: "Finish report", Status: TodoStatus_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: false,
		},
		{
			name:    "valid-todo-done",
			todo:    Todo{Title: "Finish report", Status: TodoStatus_DONE, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: false,
		},
		{
			name:    "empty-title",
			todo:    Todo{Title: "", Status: TodoStatus_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "title cannot be empty",
		},
		{
			name:    "title-too-short",
			todo:    Todo{Title: "Hi", Status: TodoStatus_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "title must be between 3 and 200 characters",
		},
		{
			name:    "title-too-long",
			todo:    Todo{Title: strings.Repeat("a", 201), Status: TodoStatus_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "title must be between 3 and 200 characters",
		},
		{
			name:    "empty-due-date",
			todo:    Todo{Title: "Finish report", Status: TodoStatus_OPEN, DueDate: time.Time{}},
			now:     now,
			wantErr: true,
			errMsg:  "due_date cannot be empty",
		},
		{
			name:    "due-date-more-than-48h-in-the-past",
			todo:    Todo{Title: "Finish report", Status: TodoStatus_OPEN, DueDate: now.Add(-49 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "due_date cannot be more than 48 hours in the past",
		},
		{
			name:    "invalid-status",
			todo:    Todo{Title: "Finish report", Status: "IN_PROGRESS", DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "status must be either OPEN or DONE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
