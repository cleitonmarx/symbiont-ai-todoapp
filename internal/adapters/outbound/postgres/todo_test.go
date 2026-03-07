package postgres

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

func TestTodoRepository_CreateTodo(t *testing.T) {
	t.Parallel()

	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedDueDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	openTodo := todo.Todo{
		ID:        fixedUUID,
		Title:     "My new todo",
		Status:    todo.Status_OPEN,
		DueDate:   fixedDueDate,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		td              todo.Todo
		expectedErr     error
	}{
		"success": {
			td: openTodo,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO todos (id,title,status,due_date,embedding,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)").
					WithArgs(
						openTodo.ID,
						openTodo.Title,
						openTodo.Status,
						openTodo.DueDate,
						pgvector.NewVector(toFloat32Truncated(openTodo.Embedding)),
						openTodo.CreatedAt,
						openTodo.UpdatedAt,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedErr: nil,
		},
		"database-error": {
			td: openTodo,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO todos (id,title,status,due_date,embedding,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)").
					WithArgs(
						openTodo.ID,
						openTodo.Title,
						openTodo.Status,
						openTodo.DueDate,
						pgvector.NewVector(toFloat32Truncated(openTodo.Embedding)),
						openTodo.CreatedAt,
						openTodo.UpdatedAt,
					).
					WillReturnError(errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.setExpectations(mock)

			repo := NewTodoRepository(db)
			gotErr := repo.CreateTodo(t.Context(), tt.td)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTodoRepository_GetTodo(t *testing.T) {
	t.Parallel()

	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedDueDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	openTodo := todo.Todo{
		ID:        fixedUUID,
		Title:     "My todo",
		Status:    todo.Status_OPEN,
		DueDate:   fixedDueDate,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		id              uuid.UUID
		expectedTodo    todo.Todo
		expectedFound   bool
		expectedErr     bool
	}{
		"success": {
			id: fixedUUID,
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						openTodo.ID,
						openTodo.Title,
						openTodo.Status,
						openTodo.DueDate,
						openTodo.CreatedAt,
						openTodo.UpdatedAt,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE id = $1").
					WithArgs(fixedUUID).
					WillReturnRows(rows)
			},
			expectedTodo:  openTodo,
			expectedFound: true,
		},
		"not-found": {
			id: fixedUUID,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE id = $1").
					WithArgs(fixedUUID).
					WillReturnError(sql.ErrNoRows)
			},
			expectedTodo: todo.Todo{},
		},
		"database-error": {
			id: fixedUUID,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE id = $1").
					WithArgs(fixedUUID).
					WillReturnError(errors.New("database error"))
			},
			expectedTodo: todo.Todo{},
			expectedErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.setExpectations(mock)

			repo := NewTodoRepository(db)
			got, gotFound, gotErr := repo.GetTodo(t.Context(), tt.id)
			if tt.expectedErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.expectedFound, gotFound)
				assert.Equal(t, tt.expectedTodo, got)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTodoRepository_UpdateTodo(t *testing.T) {
	t.Parallel()

	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedDueDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	doneTodo := todo.Todo{
		ID:        fixedUUID,
		Title:     "Updated todo",
		Status:    todo.Status_DONE,
		DueDate:   fixedDueDate,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		td              todo.Todo
		expectedErr     error
	}{
		"success": {
			td: doneTodo,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE todos SET title = $1, status = $2, due_date = $3, embedding = $4, updated_at = $5 WHERE id = $6").
					WithArgs(
						doneTodo.Title,
						doneTodo.Status,
						doneTodo.DueDate,
						pgvector.NewVector(toFloat32Truncated(doneTodo.Embedding)),
						doneTodo.UpdatedAt,
						doneTodo.ID,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		"database-error": {
			td: doneTodo,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE todos SET title = $1, status = $2, due_date = $3, embedding = $4, updated_at = $5 WHERE id = $6").
					WithArgs(
						doneTodo.Title,
						doneTodo.Status,
						doneTodo.DueDate,
						pgvector.NewVector(toFloat32Truncated(doneTodo.Embedding)),
						doneTodo.UpdatedAt,
						doneTodo.ID,
					).
					WillReturnError(errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.setExpectations(mock)

			repo := NewTodoRepository(db)
			gotErr := repo.UpdateTodo(t.Context(), tt.td)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTodoRepository_ListTodos(t *testing.T) {
	t.Parallel()

	fixedUUID1 := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedUUID2 := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedUUID3 := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedDueDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		page            int
		pageSize        int
		opts            []todo.ListOption
		expectedTodos   []todo.Todo
		expectedHasMore bool
		expectedErr     bool
	}{
		"success": {
			page:     1,
			pageSize: 10,
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID1,
						"Todo 1",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID2,
						"Todo 2",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos ORDER BY due_date ASC LIMIT 11 OFFSET 0").
					WillReturnRows(rows)
			},
			expectedTodos: []todo.Todo{
				{ID: fixedUUID1, Title: "Todo 1", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
				{ID: fixedUUID2, Title: "Todo 2", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"database-error": {
			page:     1,
			pageSize: 10,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos ORDER BY due_date ASC LIMIT 11 OFFSET 0").
					WillReturnError(errors.New("database error"))
			},
			expectedTodos:   nil,
			expectedHasMore: false,
			expectedErr:     true,
		},
		"pagination-page-2": {
			page:     2,
			pageSize: 10,
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID3,
						"Todo 3",
						todo.Status_DONE,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos ORDER BY due_date ASC LIMIT 11 OFFSET 10").
					WillReturnRows(rows)
			},
			expectedTodos: []todo.Todo{
				{ID: fixedUUID3, Title: "Todo 3", Status: todo.Status_DONE, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"invalid-page": {
			page:     0,
			pageSize: 10,
			setExpectations: func(mock sqlmock.Sqlmock) {
			},
			expectedTodos:   nil,
			expectedHasMore: false,
			expectedErr:     true,
		},
		"invalid-page-size": {
			page:     1,
			pageSize: 0,
			setExpectations: func(mock sqlmock.Sqlmock) {
			},
			expectedTodos:   nil,
			expectedHasMore: false,
			expectedErr:     true,
		},
		"has-more-results": {
			page:     1,
			pageSize: 2,
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID1,
						"Todo 1",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID2,
						"Todo 2",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID3,
						"Todo 3",
						todo.Status_DONE,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos ORDER BY due_date ASC LIMIT 3 OFFSET 0").
					WillReturnRows(rows)
			},
			expectedTodos: []todo.Todo{
				{ID: fixedUUID1, Title: "Todo 1", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
				{ID: fixedUUID2, Title: "Todo 2", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: true,
			expectedErr:     false,
		},
		"filter-by-status": {
			page:     1,
			pageSize: 10,
			opts: []todo.ListOption{
				todo.WithStatus(todo.Status_DONE),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID3,
						"Todo 3",
						todo.Status_DONE,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE status = $1 ORDER BY due_date ASC LIMIT 11 OFFSET 0").
					WithArgs(todo.Status_DONE).
					WillReturnRows(rows)
			},
			expectedTodos: []todo.Todo{
				{ID: fixedUUID3, Title: "Todo 3", Status: todo.Status_DONE, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"invalid-status-filter": {
			page:     1,
			pageSize: 10,
			opts: []todo.ListOption{
				todo.WithStatus("IN_PROGRESS"),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
			},
			expectedTodos:   nil,
			expectedHasMore: false,
			expectedErr:     true,
		},
		"filter-by-embedding": {
			page:     1,
			pageSize: 10,
			opts: []todo.ListOption{
				todo.WithEmbedding([]float64{0.1, 0.2, 0.3}),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID2,
						"Todo 2",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE (embedding <=> $1) < 0.5 AND set_config('hnsw.ef_search', '400', true) IS NOT NULL ORDER BY due_date ASC LIMIT 11 OFFSET 0").
					WithArgs(
						pgvector.NewVector([]float32{0.1, 0.2, 0.3}),
					).
					WillReturnRows(rows)
			},
			expectedTodos: []todo.Todo{
				{ID: fixedUUID2, Title: "Todo 2", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"filter-by-title-contains": {
			page:     1,
			pageSize: 10,
			opts: []todo.ListOption{
				todo.WithTitleContains("report"),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID1,
						"Finish report",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE title ILIKE $1 ORDER BY due_date ASC LIMIT 11 OFFSET 0").
					WithArgs("%report%").
					WillReturnRows(rows)
			},
			expectedTodos: []todo.Todo{
				{ID: fixedUUID1, Title: "Finish report", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
		},
		"filter-by-due-date-range": {
			page:     1,
			pageSize: 10,
			opts: []todo.ListOption{
				todo.WithDueDateRange(
					time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
				),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID2,
						"Todo 2",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE (due_date >= $1 AND due_date <= $2) ORDER BY due_date ASC LIMIT 11 OFFSET 0").
					WithArgs(
						time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
						time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
					).
					WillReturnRows(rows)
			},
			expectedTodos: []todo.Todo{
				{ID: fixedUUID2, Title: "Todo 2", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"sort-by-createdat-asc": {
			page:     1,
			pageSize: 10,
			opts: []todo.ListOption{
				todo.WithSortBy("createdAtAsc"),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID2,
						"Todo 2",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID1,
						"Todo 1",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos ORDER BY created_at ASC LIMIT 11 OFFSET 0").
					WillReturnRows(rows)
			},
			expectedTodos: []todo.Todo{
				{ID: fixedUUID2, Title: "Todo 2", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
				{ID: fixedUUID1, Title: "Todo 1", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"sort-by-similarity": {
			page:     1,
			pageSize: 10,
			opts: []todo.ListOption{
				todo.WithEmbedding([]float64{0.1, 0.2, 0.3}),
				todo.WithSortBy("similarityAsc"),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID2,
						"Todo 2",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID1,
						"Todo 1",
						todo.Status_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE (embedding <=> $1) < 0.5 AND set_config('hnsw.ef_search', '400', true) IS NOT NULL ORDER BY embedding <=> $2 ASC LIMIT 11 OFFSET 0").
					WithArgs(
						pgvector.NewVector([]float32{0.1, 0.2, 0.3}),
						pgvector.NewVector([]float32{0.1, 0.2, 0.3}),
					).
					WillReturnRows(rows)
			},
			expectedTodos: []todo.Todo{
				{ID: fixedUUID2, Title: "Todo 2", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
				{ID: fixedUUID1, Title: "Todo 1", Status: todo.Status_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"no-embedding-for-similarity-sort": {
			page:     1,
			pageSize: 10,
			opts: []todo.ListOption{
				todo.WithSortBy("similarityAsc"),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
			},
			expectedTodos:   nil,
			expectedHasMore: false,
			expectedErr:     true,
		},
		"invalid-sort-by": {
			page:     1,
			pageSize: 10,
			opts: []todo.ListOption{
				todo.WithSortBy("invalidSort"),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
			},
			expectedTodos:   nil,
			expectedHasMore: false,
			expectedErr:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.setExpectations(mock)

			repo := NewTodoRepository(db)
			got, hasMore, gotErr := repo.ListTodos(t.Context(), tt.page, tt.pageSize, tt.opts...)
			if tt.expectedErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.expectedTodos, got)
				assert.Equal(t, tt.expectedHasMore, hasMore)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTodoRepository_DeleteTodo(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM todos WHERE id = $1").
					WithArgs(id).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			err: false,
		},
		"db-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM todos WHERE id = $1").
					WithArgs(id).
					WillReturnError(errors.New("db error"))
			},
			err: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() //nolint:errcheck

			tt.expect(mock)

			repo := NewTodoRepository(db)
			gotErr := repo.DeleteTodo(t.Context(), id)

			if tt.err {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
