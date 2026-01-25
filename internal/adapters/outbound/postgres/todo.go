package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
)

var (
	todoFields = []string{
		"id",
		"title",
		"status",
		"due_date",
		"created_at",
		"updated_at",
	}
)

// TodoRepository is an in-memory implementation of domain.Repository for Todos.
type TodoRepository struct {
	sb squirrel.StatementBuilderType
}

// NewTodoRepository creates a new instance of TodoRepository.
func NewTodoRepository(br squirrel.BaseRunner) TodoRepository {
	return TodoRepository{
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(br),
	}
}

// ListTodos lists todos with pagination and optional filters.
func (tr TodoRepository) ListTodos(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) ([]domain.Todo, bool, error) {
	spanCtx, span := tracing.Start(ctx, trace.WithAttributes(
		attribute.Int("page", page),
		attribute.Int("pageSize", pageSize),
	))
	defer span.End()

	qry := tr.sb.
		Select(
			todoFields...,
		).From("todos").
		OrderBy("created_at DESC").
		Limit(uint64(pageSize + 1)). // fetch one extra to determine if there's more
		Offset(uint64((page - 1) * pageSize))

	params := &domain.ListTodosParams{}
	for _, opt := range opts {
		opt(params)
	}

	if params.Status != nil {
		qry = qry.Where(squirrel.Eq{"status": *params.Status})
	}

	rows, err := qry.QueryContext(spanCtx)
	if tracing.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	defer rows.Close() //nolint:errcheck

	var todos []domain.Todo
	for rows.Next() {
		var todo domain.Todo
		err := rows.Scan(
			&todo.ID,
			&todo.Title,
			&todo.Status,
			&todo.DueDate,
			&todo.CreatedAt,
			&todo.UpdatedAt,
		)
		if tracing.RecordErrorAndStatus(span, err) {
			return nil, false, err
		}
		todos = append(todos, todo)
	}

	if err := rows.Err(); tracing.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}

	if len(todos) > pageSize {
		todos = todos[:pageSize]
		return todos, true, nil
	}
	return todos, false, nil
}

// CreateTodo creates a new todo.
func (tr TodoRepository) CreateTodo(ctx context.Context, todo domain.Todo) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	_, err := tr.sb.
		Insert("todos").
		Columns(
			todoFields...,
		).
		Values(
			todo.ID,
			todo.Title,
			todo.Status,
			todo.DueDate,
			todo.CreatedAt,
			todo.UpdatedAt,
		).
		ExecContext(spanCtx)

	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// UpdateTodo updates an existing todo.
func (tr TodoRepository) UpdateTodo(ctx context.Context, todo domain.Todo) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	_, err := tr.sb.
		Update("todos").
		Set("title", todo.Title).
		Set("status", todo.Status).
		Set("due_date", todo.DueDate).
		Set("updated_at", todo.UpdatedAt).
		Where(squirrel.Eq{"id": todo.ID}).
		ExecContext(spanCtx)

	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// DeleteTodo deletes a todo by its ID.
func (tr TodoRepository) DeleteTodo(ctx context.Context, id uuid.UUID) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	_, err := tr.sb.
		Delete("todos").
		Where(squirrel.Eq{"id": id}).
		ExecContext(spanCtx)

	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// GetTodo retrieves a todo by its ID.
func (tr TodoRepository) GetTodo(ctx context.Context, id uuid.UUID) (domain.Todo, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	var todo domain.Todo
	err := tr.sb.
		Select(
			todoFields...,
		).
		From("todos").
		Where(squirrel.Eq{"id": id}).
		QueryRowContext(spanCtx).
		Scan(
			&todo.ID,
			&todo.Title,
			&todo.Status,
			&todo.DueDate,
			&todo.CreatedAt,
			&todo.UpdatedAt,
		)

	if tracing.RecordErrorAndStatus(span, err) {
		if err == sql.ErrNoRows {
			return domain.Todo{}, fmt.Errorf("todo with id %s not found", id)
		}
		return domain.Todo{}, err
	}

	return todo, nil
}

// InitTodoRepository is a Symbiont initializer for TodoRepository.
type InitTodoRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the TodoRepository in the dependency container.
func (tr InitTodoRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.TodoRepository](NewTodoRepository(tr.DB))
	return ctx, nil
}
