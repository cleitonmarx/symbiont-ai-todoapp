package domain

import (
	"testing"
	"time"

	"strings"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
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

func TestListTodoOptions_WithOptions(t *testing.T) {
	tests := map[string]struct {
		opts []ListTodoOptions
		want ListTodosParams
	}{
		"with-status-and-embedding": {
			opts: []ListTodoOptions{WithStatus(TodoStatus_DONE), WithEmbedding([]float64{0.1, 0.2, 0.3})},
			want: ListTodosParams{Status: common.Ptr(TodoStatus_DONE), Embedding: []float64{0.1, 0.2, 0.3}},
		},
		"with-status-only": {
			opts: []ListTodoOptions{WithStatus(TodoStatus_DONE)},
			want: ListTodosParams{Status: common.Ptr(TodoStatus_DONE)},
		},
		"with-embedding-only": {
			opts: []ListTodoOptions{WithEmbedding([]float64{0.1, 0.2, 0.3})},
			want: ListTodosParams{Embedding: []float64{0.1, 0.2, 0.3}},
		},
		"with-due-date-range-only": {
			opts: []ListTodoOptions{
				WithDueDateRange(
					time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 7, 20, 0, 0, 0, 0, time.UTC),
				),
			},
			want: ListTodosParams{
				DueAfter:  common.Ptr(time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC)),
				DueBefore: common.Ptr(time.Date(2024, 7, 20, 0, 0, 0, 0, time.UTC)),
			},
		},
		"with-sort-by-only": {
			opts: []ListTodoOptions{
				WithSortBy("dueDateDesc"),
			},
			want: ListTodosParams{
				SortBy: &TodoSortBy{Field: "dueDate", Direction: "DESC"},
			},
		},
		"with-multiple-options": {
			opts: []ListTodoOptions{
				WithStatus(TodoStatus_OPEN),
				WithEmbedding([]float64{0.4, 0.5, 0.6}),
				WithDueDateRange(
					time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC),
				),
				WithSortBy("createdAtAsc"),
			},
			want: ListTodosParams{
				Status:    common.Ptr(TodoStatus_OPEN),
				Embedding: []float64{0.4, 0.5, 0.6},
				DueAfter:  common.Ptr(time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)),
				DueBefore: common.Ptr(time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC)),
				SortBy:    &TodoSortBy{Field: "createdAt", Direction: "ASC"},
			},
		},
		"with-no-options": {
			opts: nil,
			want: ListTodosParams{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			params := &ListTodosParams{}
			for _, opt := range tt.opts {
				opt(params)
			}
			assert.Equal(t, tt.want, *params)
		})
	}
}

func TestTodoSortBy_Validate(t *testing.T) {
	tests := map[string]struct {
		sortField     string
		wantField     string
		wantDirection string
		wantErr       bool
	}{
		"valid-createdAt-asc": {
			sortField:     "createdAtAsc",
			wantField:     "created_at",
			wantDirection: "ASC",
			wantErr:       false,
		},
		"valid-createdAt-desc": {
			sortField:     "createdAtDesc",
			wantField:     "created_at",
			wantDirection: "DESC",
			wantErr:       false,
		},
		"valid-dueDate-asc": {
			sortField:     "dueDateAsc",
			wantField:     "due_date",
			wantDirection: "ASC",
			wantErr:       false,
		},
		"valid-dueDate-desc": {
			sortField:     "dueDateDesc",
			wantField:     "due_date",
			wantDirection: "DESC",
			wantErr:       false,
		},
		"invalid-field": {
			sortField: "priorityAsc",
			wantErr:   true,
		},
		"invalid-direction": {
			sortField: "createdAt",
			wantErr:   true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := ListTodosParams{}
			WithSortBy(tt.sortField)(&p)
			gotErr := p.SortBy.Validate()
			if tt.wantErr {
				assert.Error(t, gotErr)
				return
			}
			assert.NoError(t, gotErr)
			assert.Equal(t, tt.wantField, p.SortBy.Field)
			assert.Equal(t, tt.wantDirection, p.SortBy.Direction)
		})
	}
}
