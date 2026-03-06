package todo

import (
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/google/uuid"
)

// Todo represents a todo item in the system.
type Todo struct {
	ID        uuid.UUID
	Title     string
	DueDate   time.Time
	Status    Status
	Embedding []float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate verifies the Todo fields satisfy domain constraints.
func (t Todo) Validate(now time.Time) error {
	if t.Title == "" {
		return core.NewValidationErr("title cannot be empty")
	}
	if len(t.Title) < 3 || len(t.Title) > 200 {
		err := core.NewValidationErr("title must be between 3 and 200 characters")
		return err
	}
	if t.DueDate.IsZero() {
		return core.NewValidationErr("due_date cannot be empty")
	}
	if err := t.Status.Validate(); err != nil {
		return err
	}

	return nil
}
