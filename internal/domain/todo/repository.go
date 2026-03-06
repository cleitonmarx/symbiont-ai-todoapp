package todo

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for interacting with todo items in storage.
type Repository interface {
	// ListTodos retrieves a list of todo items with pagination support.
	ListTodos(ctx context.Context, page int, pageSize int, opts ...ListOption) ([]Todo, bool, error)

	// CreateTodo creates a new todo item.
	CreateTodo(ctx context.Context, todo Todo) error

	// UpdateTodo updates an existing todo item.
	UpdateTodo(ctx context.Context, todo Todo) error

	// DeleteTodo removes a todo item by ID.
	DeleteTodo(ctx context.Context, id uuid.UUID) error

	// GetTodo retrieves one todo item by ID.
	GetTodo(ctx context.Context, id uuid.UUID) (Todo, bool, error)
}
