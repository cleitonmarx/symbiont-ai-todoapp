package postgres

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
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

// TodoRepository implements the todo.Repository interface using PostgreSQL as the storage backend.
type TodoRepository struct {
	sb sq.StatementBuilderType
}

// NewTodoRepository creates a new instance of TodoRepository.
func NewTodoRepository(br sq.BaseRunner) TodoRepository {
	return TodoRepository{
		sb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(br),
	}
}

// ListTodos lists todos with pagination and optional filters.
func (tr TodoRepository) ListTodos(ctx context.Context, page int, pageSize int, opts ...todo.ListOption) ([]todo.Todo, bool, error) {
	spanCtx, span := telemetry.Start(ctx, trace.WithAttributes(
		attribute.Int("page", page),
		attribute.Int("pageSize", pageSize),
	))
	defer span.End()

	if pageSize <= 0 {
		return nil, false, core.NewValidationErr("page_size must be greater than 0")
	}
	if page <= 0 {
		return nil, false, core.NewValidationErr("page must be greater than 0")
	}

	qry := tr.sb.
		Select(
			todoFields...,
		).From("todos").
		Limit(uint64(pageSize + 1)). // fetch one extra to determine if there's more
		Offset(uint64((page - 1) * pageSize))

	params := &todo.ListParams{}
	for _, opt := range opts {
		opt(params)
	}

	if params.Status != nil {
		if err := params.Status.Validate(); err != nil {
			return nil, false, err
		}
		qry = qry.Where(sq.Eq{"status": *params.Status})
	}

	if len(params.Embedding) > 0 {
		qry = qry.
			Where(sq.Expr(
				"(embedding <=> ?) < 0.5",
				pgvector.NewVector(toFloat32Truncated(params.Embedding)),
			)).
			Where(sq.Expr(
				"set_config('hnsw.ef_search', '400', true) IS NOT NULL",
			))
	}

	if params.TitleContains != nil {
		qry = qry.Where(sq.ILike{"title": "%" + *params.TitleContains + "%"})
	}

	if params.DueAfter != nil && params.DueBefore != nil {
		qry = qry.Where(sq.And{
			sq.GtOrEq{"due_date": *params.DueAfter},
			sq.LtOrEq{"due_date": *params.DueBefore},
		})
	}

	qry, err := applySort(qry, params)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}

	rows, err := qry.QueryContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	defer rows.Close() //nolint:errcheck

	var todos []todo.Todo
	for rows.Next() {
		var td todo.Todo
		err := rows.Scan(
			&td.ID,
			&td.Title,
			&td.Status,
			&td.DueDate,
			&td.CreatedAt,
			&td.UpdatedAt,
		)
		if telemetry.RecordErrorAndStatus(span, err) {
			return nil, false, err
		}
		todos = append(todos, td)
	}

	if err := rows.Err(); telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}

	if len(todos) > pageSize {
		todos = todos[:pageSize]
		return todos, true, nil
	}
	return todos, false, nil
}

// applySort applies sorting to the given squirrel SelectBuilder based on the provided ListTodosParams.
func applySort(qry sq.SelectBuilder, params *todo.ListParams) (sq.SelectBuilder, error) {
	if params.SortBy == nil {
		return qry.OrderBy("due_date ASC"), nil
	}

	if err := params.SortBy.Validate(); err != nil {
		return qry, err
	}

	if params.SortBy.Field == "similarity" && len(params.Embedding) > 0 {
		return qry.OrderByClause(sq.Expr(
			"embedding <=> ? "+params.SortBy.Direction,
			pgvector.NewVector(toFloat32Truncated(params.Embedding)),
		)), nil
	} else if params.SortBy.Field == "similarity" && len(params.Embedding) == 0 {
		return qry, core.NewValidationErr("embedding must be provided for similarity sorting")
	}

	orderClause := params.SortBy.Field + " " + params.SortBy.Direction
	return qry.OrderBy(orderClause), nil
}

// CreateTodo creates a new todo.
func (tr TodoRepository) CreateTodo(ctx context.Context, td todo.Todo) error {
	spanCtx, span := telemetry.Start(ctx)
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
			td.ID,
			td.Title,
			td.Status,
			td.DueDate,
			pgvector.NewVector(toFloat32Truncated(td.Embedding)),
			td.CreatedAt,
			td.UpdatedAt,
		).
		ExecContext(spanCtx)

	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// UpdateTodo updates an existing todo.
func (tr TodoRepository) UpdateTodo(ctx context.Context, td todo.Todo) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	_, err := tr.sb.
		Update("todos").
		Set("title", td.Title).
		Set("status", td.Status).
		Set("due_date", td.DueDate).
		Set("embedding", pgvector.NewVector(toFloat32Truncated(td.Embedding))).
		Set("updated_at", td.UpdatedAt).
		Where(sq.Eq{"id": td.ID}).
		ExecContext(spanCtx)

	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// DeleteTodo deletes a todo by its ID.
func (tr TodoRepository) DeleteTodo(ctx context.Context, id uuid.UUID) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	_, err := tr.sb.
		Delete("todos").
		Where(sq.Eq{"id": id}).
		ExecContext(spanCtx)

	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// GetTodo retrieves a todo by its ID.
func (tr TodoRepository) GetTodo(ctx context.Context, id uuid.UUID) (todo.Todo, bool, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	var td todo.Todo
	err := tr.sb.
		Select(
			todoFields...,
		).
		From("todos").
		Where(sq.Eq{"id": id}).
		QueryRowContext(spanCtx).
		Scan(
			&td.ID,
			&td.Title,
			&td.Status,
			&td.DueDate,
			&td.CreatedAt,
			&td.UpdatedAt,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return todo.Todo{}, false, nil
	}

	if telemetry.RecordErrorAndStatus(span, err) {
		return todo.Todo{}, false, err
	}

	return td, true, nil
}

// toFloat32Truncated converts a slice of float64 to a slice of float32, truncating to 768 dimensions if necessary.
func toFloat32Truncated(input []float64) []float32 {
	f32 := make([]float32, len(input))
	for i, v := range input {
		f32[i] = float32(v)
	}
	if len(f32) > 768 {
		f32 = f32[:768]
	}
	return f32
}
