package todo

import (
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
)

// SortBy represents sorting criteria for listing todos.
type SortBy struct {
	Field     string
	Direction string
}

// Validate checks if the SortBy fields are valid.
func (s *SortBy) Validate() error {
	allowedFields := map[string]string{
		"createdAt":  "created_at",
		"dueDate":    "due_date",
		"similarity": "similarity",
	}
	val, ok := allowedFields[s.Field]
	if !ok {
		return core.NewValidationErr("invalid sort field: " + s.Field)
	}
	if s.Direction != "ASC" && s.Direction != "DESC" {
		return core.NewValidationErr("invalid sort direction: " + s.Direction)
	}
	s.Field = val
	return nil
}

// ListParams represents the parameters for listing todo items.
type ListParams struct {
	Status        *Status
	Embedding     []float64
	TitleContains *string
	DueAfter      *time.Time
	DueBefore     *time.Time
	SortBy        *SortBy
}

// ListOption defines a function type for modifying ListParams.
type ListOption func(*ListParams)

// WithStatus filters todos by their status.
func WithStatus(status Status) ListOption {
	return func(params *ListParams) {
		params.Status = &status
	}
}

// WithEmbedding filters todos by embedding similarity to the provided vector.
func WithEmbedding(embedding []float64) ListOption {
	return func(params *ListParams) {
		params.Embedding = embedding
	}
}

// WithTitleContains filters todos whose title contains the specified substring.
func WithTitleContains(substring string) ListOption {
	return func(params *ListParams) {
		params.TitleContains = &substring
	}
}

// WithDueDateRange filters todos by a due date range.
func WithDueDateRange(dueAfter, dueBefore time.Time) ListOption {
	return func(params *ListParams) {
		params.DueAfter = &dueAfter
		params.DueBefore = &dueBefore
	}
}

// WithSortBy sets sorting criteria for listing todos.
func WithSortBy(sort string) ListOption {
	return func(params *ListParams) {
		if after, ok := strings.CutSuffix(sort, "Desc"); ok {
			params.SortBy = &SortBy{Field: after, Direction: "DESC"}
			return
		}
		if after, ok := strings.CutSuffix(sort, "Asc"); ok {
			params.SortBy = &SortBy{Field: after, Direction: "ASC"}
			return
		}
		params.SortBy = &SortBy{Field: sort, Direction: ""}
	}
}
