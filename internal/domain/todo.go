package domain

import (
	"context"
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
	ID        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	DueDate   time.Time  `json:"due_date"`
	Status    TodoStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ListTodosParams represents the parameters for listing todo items.
type ListTodosParams struct {
	Status *TodoStatus
}

// ListTodoOptions defines a function type for modifying ListTodosParams.
type ListTodoOptions func(*ListTodosParams)

// WithStatus is a ListTodoOptions that filters todos by their status.
func WithStatus(status TodoStatus) ListTodoOptions {
	return func(params *ListTodosParams) {
		params.Status = &status
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
	GetTodo(ctx context.Context, id uuid.UUID) (Todo, error)
}
