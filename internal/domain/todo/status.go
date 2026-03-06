package todo

import "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"

// Status represents the status of a todo item.
type Status string

const (
	// Status_OPEN indicates that the todo item is open and not yet completed.
	Status_OPEN Status = "OPEN"
	// Status_DONE indicates that the todo item has been completed.
	Status_DONE Status = "DONE"
)

// Validate checks if the Status is valid.
func (s Status) Validate() error {
	if s != Status_OPEN && s != Status_DONE {
		return core.NewValidationErr("status must be either OPEN or DONE")
	}
	return nil
}
