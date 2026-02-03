package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TodoStatus represents the status of a todo item.
type TodoStatus string

const (
	// TodoStatus_OPEN indicates that the todo item is open and not yet completed.
	TodoStatus_OPEN TodoStatus = "OPEN"
	// TodoStatus_DONE indicates that the todo item has been completed.
	TodoStatus_DONE TodoStatus = "DONE"
)

// Todo represents a todo item in the system.
type Todo struct {
	ID        uuid.UUID
	Title     string
	DueDate   time.Time
	Status    TodoStatus
	Embedding []float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (t Todo) Validate(now time.Time) error {
	if t.Title == "" {
		return NewValidationErr("title cannot be empty")
	}
	if len(t.Title) < 3 || len(t.Title) > 200 {
		err := NewValidationErr("title must be between 3 and 200 characters")
		return err
	}
	if t.DueDate.IsZero() {
		return NewValidationErr("due_date cannot be empty")
	}
	if t.DueDate.Truncate(24 * time.Hour).Before(now.Add(-48 * time.Hour).Truncate(24 * time.Hour)) {
		return NewValidationErr("due_date cannot be more than 48 hours in the past")
	}
	if t.Status != TodoStatus_OPEN && t.Status != TodoStatus_DONE {
		return NewValidationErr("status must be either OPEN or DONE")
	}

	return nil
}

// ToLLMInput formats the todo item as a string suitable for LLM input.
func (t Todo) ToLLMInput() string {
	return fmt.Sprintf("ID: %s | Title: %s | Due Date: %s | Status: %s", t.ID.String(), t.Title, t.DueDate.Format(time.DateOnly), t.Status)
}

// ListTodosParams represents the parameters for listing todo items.
type ListTodosParams struct {
	Status    *TodoStatus
	Embedding []float64
}

// ListTodoOptions defines a function type for modifying ListTodosParams.
type ListTodoOptions func(*ListTodosParams)

// WithStatus is a ListTodoOptions that filters todos by their status.
func WithStatus(status TodoStatus) ListTodoOptions {
	return func(params *ListTodosParams) {
		params.Status = &status
	}
}

// WithEmbedding is a ListTodoOptions that filters todos by their embedding similarity to the provided embedding.
func WithEmbedding(embedding []float64) ListTodoOptions {
	return func(params *ListTodosParams) {
		params.Embedding = embedding
	}
}

// TodoRepository defines the interface for interacting with todo items in the data store.
type TodoRepository interface {
	// ListTodos retrieves a list of todo items with pagination support.
	ListTodos(ctx context.Context, page int, pageSize int, opts ...ListTodoOptions) ([]Todo, bool, error)

	// CreateTodo creates a new todo item with the given title.
	CreateTodo(ctx context.Context, todo Todo) error

	// UpdateTodo updates an existing todo item identified by id with the provided fields.
	UpdateTodo(ctx context.Context, todo Todo) error

	// DeleteTodo removes a todo item identified by id from the data store.
	DeleteTodo(ctx context.Context, id uuid.UUID) error

	// GetTodo retrieves a todo item by its unique identifier.
	GetTodo(ctx context.Context, id uuid.UUID) (Todo, bool, error)
}
