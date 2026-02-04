package postgres

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

// TodoRepository implements the domain.TodoRepository interface using PostgreSQL as the storage backend.
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

	if pageSize <= 0 {
		return nil, false, domain.NewValidationErr("page_size must be greater than 0")
	}
	if page <= 0 {
		return nil, false, domain.NewValidationErr("page must be greater than 0")
	}

	qry := tr.sb.
		Select(
			todoFields...,
		).From("todos").
		Limit(uint64(pageSize + 1)). // fetch one extra to determine if there's more
		Offset(uint64((page - 1) * pageSize))

	params := &domain.ListTodosParams{}
	for _, opt := range opts {
		opt(params)
	}

	if params.Status != nil {
		qry = qry.Where(squirrel.Eq{"status": *params.Status})
	}

	if len(params.Embedding) > 0 {
		qry = qry.Where(squirrel.Expr(
			"(embedding <=> ?) < 0.5",
			pgvector.NewVector(toFloat32Truncated(params.Embedding)),
		))
	}
	if params.DueAfter != nil && params.DueBefore != nil {
		qry = qry.Where(squirrel.And{
			squirrel.GtOrEq{"due_date": *params.DueAfter},
			squirrel.LtOrEq{"due_date": *params.DueBefore},
		})
	}
	qry, err := applySort(qry, params)
	if tracing.RecordErrorAndStatus(span, err) {
		return nil, false, err
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

// applySort applies sorting to the given squirrel SelectBuilder based on the provided ListTodosParams.
func applySort(qry squirrel.SelectBuilder, params *domain.ListTodosParams) (squirrel.SelectBuilder, error) {
	if params.SortBy == nil {
		return qry.OrderBy("created_at DESC"), nil
	}

	if err := params.SortBy.Validate(); err != nil {
		return qry, err
	}

	if params.SortBy.Field == "similarity" && len(params.Embedding) > 0 {
		return qry.OrderByClause(squirrel.Expr(
			"embedding <#> ? "+params.SortBy.Direction,
			pgvector.NewVector(toFloat32Truncated(params.Embedding)),
		)), nil
	} else if params.SortBy.Field == "similarity" && len(params.Embedding) == 0 {
		return qry, domain.NewValidationErr("embedding must be provided for similarity sorting")
	}

	orderClause := params.SortBy.Field + " " + params.SortBy.Direction
	return qry.OrderBy(orderClause), nil
}

// CreateTodo creates a new todo.
func (tr TodoRepository) CreateTodo(ctx context.Context, todo domain.Todo) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	_, err := tr.sb.
		Insert("todos").
		Columns(
			"id",
			"title",
			"status",
			"due_date",
			"embedding",
			"created_at",
			"updated_at",
		).
		Values(
			todo.ID,
			todo.Title,
			todo.Status,
			todo.DueDate,
			pgvector.NewVector(toFloat32Truncated(todo.Embedding)),
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
		Set("embedding", pgvector.NewVector(toFloat32Truncated(todo.Embedding))).
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
func (tr TodoRepository) GetTodo(ctx context.Context, id uuid.UUID) (domain.Todo, bool, error) {
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
			return domain.Todo{}, false, nil
		}
		return domain.Todo{}, false, err
	}

	return todo, true, nil
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

func toFloat32Truncated(input []float64) []float32 {
	f32 := make([]float32, len(input))
	for i, v := range input {
		f32[i] = float32(v)
	}
	if len(f32) > 1536 {
		f32 = f32[:1536]
	}
	return f32
}
