package todo

import (
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/stretchr/testify/assert"
)

func TestTodo_Validate(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		todo    Todo
		now     time.Time
		wantErr bool
		errMsg  string
	}{
		"valid-todo-open": {
			todo:    Todo{Title: "Finish report", Status: Status_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: false,
		},
		"valid-todo-done": {
			todo:    Todo{Title: "Finish report", Status: Status_DONE, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: false,
		},
		"empty-title": {
			todo:    Todo{Title: "", Status: Status_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "title cannot be empty",
		},
		"title-too-short": {
			todo:    Todo{Title: "Hi", Status: Status_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "title must be between 3 and 200 characters",
		},
		"title-too-long": {
			todo:    Todo{Title: strings.Repeat("a", 201), Status: Status_OPEN, DueDate: now.Add(24 * time.Hour)},
			now:     now,
			wantErr: true,
			errMsg:  "title must be between 3 and 200 characters",
		},
		"empty-due-date": {
			todo:    Todo{Title: "Finish report", Status: Status_OPEN, DueDate: time.Time{}},
			now:     now,
			wantErr: true,
			errMsg:  "due_date cannot be empty",
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

func TestListOptions_WithOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		opts []ListOption
		want ListParams
	}{
		"with-status-and-embedding": {
			opts: []ListOption{WithStatus(Status_DONE), WithEmbedding([]float64{0.1, 0.2, 0.3})},
			want: ListParams{Status: common.Ptr(Status_DONE), Embedding: []float64{0.1, 0.2, 0.3}},
		},
		"with-status-only": {
			opts: []ListOption{WithStatus(Status_DONE)},
			want: ListParams{Status: common.Ptr(Status_DONE)},
		},
		"with-embedding-only": {
			opts: []ListOption{WithEmbedding([]float64{0.1, 0.2, 0.3})},
			want: ListParams{Embedding: []float64{0.1, 0.2, 0.3}},
		},
		"with-title-contains-only": {
			opts: []ListOption{WithTitleContains("report")},
			want: ListParams{TitleContains: common.Ptr("report")},
		},
		"with-due-date-range-only": {
			opts: []ListOption{
				WithDueDateRange(
					time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 7, 20, 0, 0, 0, 0, time.UTC),
				),
			},
			want: ListParams{
				DueAfter:  common.Ptr(time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC)),
				DueBefore: common.Ptr(time.Date(2024, 7, 20, 0, 0, 0, 0, time.UTC)),
			},
		},
		"with-sort-by-only": {
			opts: []ListOption{
				WithSortBy("dueDateDesc"),
			},
			want: ListParams{
				SortBy: &SortBy{Field: "dueDate", Direction: "DESC"},
			},
		},
		"with-multiple-options": {
			opts: []ListOption{
				WithStatus(Status_OPEN),
				WithEmbedding([]float64{0.4, 0.5, 0.6}),
				WithDueDateRange(
					time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC),
				),
				WithSortBy("createdAtAsc"),
			},
			want: ListParams{
				Status:    common.Ptr(Status_OPEN),
				Embedding: []float64{0.4, 0.5, 0.6},
				DueAfter:  common.Ptr(time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)),
				DueBefore: common.Ptr(time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC)),
				SortBy:    &SortBy{Field: "createdAt", Direction: "ASC"},
			},
		},
		"with-no-options": {
			opts: nil,
			want: ListParams{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			params := &ListParams{}
			for _, opt := range tt.opts {
				opt(params)
			}
			assert.Equal(t, tt.want, *params)
		})
	}
}

func TestSortBy_Validate(t *testing.T) {
	t.Parallel()

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
			p := ListParams{}
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

func TestStatus_Validate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status  Status
		wantErr error
	}{
		"valid-open": {
			status:  Status_OPEN,
			wantErr: nil,
		},
		"valid-done": {
			status:  Status_DONE,
			wantErr: nil,
		},
		"invalid-status": {
			status:  "INVALID",
			wantErr: core.NewValidationErr("status must be either OPEN or DONE"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotErr := tt.status.Validate()
			assert.Equal(t, tt.wantErr, gotErr)

		})
	}
}
