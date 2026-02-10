package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

func TestTodoRepository_CreateTodo(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedDueDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		ID:        fixedUUID,
		Title:     "My new todo",
		Status:    domain.TodoStatus_OPEN,
		DueDate:   fixedDueDate,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		todo            domain.Todo
		expectedErr     error
	}{
		"success": {
			todo: todo,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO todos (id,title,status,due_date,embedding,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)").
					WithArgs(
						todo.ID,
						todo.Title,
						todo.Status,
						todo.DueDate,
						pgvector.NewVector(toFloat32Truncated(todo.Embedding)),
						todo.CreatedAt,
						todo.UpdatedAt,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedErr: nil,
		},
		"database-error": {
			todo: todo,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO todos (id,title,status,due_date,embedding,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)").
					WithArgs(
						todo.ID,
						todo.Title,
						todo.Status,
						todo.DueDate,
						pgvector.NewVector(toFloat32Truncated(todo.Embedding)),
						todo.CreatedAt,
						todo.UpdatedAt,
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
			gotErr := repo.CreateTodo(context.Background(), tt.todo)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTodoRepository_GetTodo(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedDueDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		ID:        fixedUUID,
		Title:     "My todo",
		Status:    domain.TodoStatus_OPEN,
		DueDate:   fixedDueDate,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		id              uuid.UUID
		expectedTodo    domain.Todo
		expectedFound   bool
		expectedErr     bool
	}{
		"success": {
			id: fixedUUID,
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						todo.ID,
						todo.Title,
						todo.Status,
						todo.DueDate,
						todo.CreatedAt,
						todo.UpdatedAt,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE id = $1").
					WithArgs(fixedUUID).
					WillReturnRows(rows)
			},
			expectedTodo:  todo,
			expectedFound: true,
		},
		"not-found": {
			id: fixedUUID,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE id = $1").
					WithArgs(fixedUUID).
					WillReturnError(sql.ErrNoRows)
			},
			expectedTodo: domain.Todo{},
		},
		"database-error": {
			id: fixedUUID,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE id = $1").
					WithArgs(fixedUUID).
					WillReturnError(errors.New("database error"))
			},
			expectedTodo: domain.Todo{},
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
			got, gotFound, gotErr := repo.GetTodo(context.Background(), tt.id)
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
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedDueDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		ID:        fixedUUID,
		Title:     "Updated todo",
		Status:    domain.TodoStatus_DONE,
		DueDate:   fixedDueDate,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		todo            domain.Todo
		expectedErr     error
	}{
		"success": {
			todo: todo,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE todos SET title = $1, status = $2, due_date = $3, embedding = $4, updated_at = $5 WHERE id = $6").
					WithArgs(
						todo.Title,
						todo.Status,
						todo.DueDate,
						pgvector.NewVector(toFloat32Truncated(todo.Embedding)),
						todo.UpdatedAt,
						todo.ID,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		"database-error": {
			todo: todo,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE todos SET title = $1, status = $2, due_date = $3, embedding = $4, updated_at = $5 WHERE id = $6").
					WithArgs(
						todo.Title,
						todo.Status,
						todo.DueDate,
						pgvector.NewVector(toFloat32Truncated(todo.Embedding)),
						todo.UpdatedAt,
						todo.ID,
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
			gotErr := repo.UpdateTodo(context.Background(), tt.todo)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTodoRepository_ListTodos(t *testing.T) {
	fixedUUID1 := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedUUID2 := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedUUID3 := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fixedDueDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		page            int
		pageSize        int
		opts            []domain.ListTodoOptions
		expectedTodos   []domain.Todo
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
						domain.TodoStatus_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID2,
						"Todo 2",
						domain.TodoStatus_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos ORDER BY due_date ASC LIMIT 11 OFFSET 0").
					WillReturnRows(rows)
			},
			expectedTodos: []domain.Todo{
				{ID: fixedUUID1, Title: "Todo 1", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
				{ID: fixedUUID2, Title: "Todo 2", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
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
						domain.TodoStatus_DONE,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos ORDER BY due_date ASC LIMIT 11 OFFSET 10").
					WillReturnRows(rows)
			},
			expectedTodos: []domain.Todo{
				{ID: fixedUUID3, Title: "Todo 3", Status: domain.TodoStatus_DONE, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
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
						domain.TodoStatus_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID2,
						"Todo 2",
						domain.TodoStatus_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID3,
						"Todo 3",
						domain.TodoStatus_DONE,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos ORDER BY due_date ASC LIMIT 3 OFFSET 0").
					WillReturnRows(rows)
			},
			expectedTodos: []domain.Todo{
				{ID: fixedUUID1, Title: "Todo 1", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
				{ID: fixedUUID2, Title: "Todo 2", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: true,
			expectedErr:     false,
		},
		"filter-by-status": {
			page:     1,
			pageSize: 10,
			opts: []domain.ListTodoOptions{
				domain.WithStatus(domain.TodoStatus_DONE),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID3,
						"Todo 3",
						domain.TodoStatus_DONE,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE status = $1 ORDER BY due_date ASC LIMIT 11 OFFSET 0").
					WithArgs(domain.TodoStatus_DONE).
					WillReturnRows(rows)
			},
			expectedTodos: []domain.Todo{
				{ID: fixedUUID3, Title: "Todo 3", Status: domain.TodoStatus_DONE, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"invalid-status-filter": {
			page:     1,
			pageSize: 10,
			opts: []domain.ListTodoOptions{
				domain.WithStatus("IN_PROGRESS"),
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
			opts: []domain.ListTodoOptions{
				domain.WithEmbedding([]float64{0.1, 0.2, 0.3}),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID2,
						"Todo 2",
						domain.TodoStatus_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE (embedding <=> $1) < 0.5 ORDER BY due_date ASC LIMIT 11 OFFSET 0").
					WithArgs(
						pgvector.NewVector([]float32{0.1, 0.2, 0.3}),
					).
					WillReturnRows(rows)
			},
			expectedTodos: []domain.Todo{
				{ID: fixedUUID2, Title: "Todo 2", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"filter-by-due-date-range": {
			page:     1,
			pageSize: 10,
			opts: []domain.ListTodoOptions{
				domain.WithDueDateRange(
					time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
				),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID2,
						"Todo 2",
						domain.TodoStatus_OPEN,
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
			expectedTodos: []domain.Todo{
				{ID: fixedUUID2, Title: "Todo 2", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"sort-by-createdat-asc": {
			page:     1,
			pageSize: 10,
			opts: []domain.ListTodoOptions{
				domain.WithSortBy("createdAtAsc"),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID2,
						"Todo 2",
						domain.TodoStatus_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID1,
						"Todo 1",
						domain.TodoStatus_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos ORDER BY created_at ASC LIMIT 11 OFFSET 0").
					WillReturnRows(rows)
			},
			expectedTodos: []domain.Todo{
				{ID: fixedUUID2, Title: "Todo 2", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
				{ID: fixedUUID1, Title: "Todo 1", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"sort-by-similarity": {
			page:     1,
			pageSize: 10,
			opts: []domain.ListTodoOptions{
				domain.WithEmbedding([]float64{0.1, 0.2, 0.3}),
				domain.WithSortBy("similarityAsc"),
			},
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(todoFields).
					AddRow(
						fixedUUID2,
						"Todo 2",
						domain.TodoStatus_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					).
					AddRow(
						fixedUUID1,
						"Todo 1",
						domain.TodoStatus_OPEN,
						fixedDueDate,
						fixedTime,
						fixedTime,
					)
				mock.ExpectQuery("SELECT id, title, status, due_date, created_at, updated_at FROM todos WHERE (embedding <=> $1) < 0.5 ORDER BY embedding <#> $2 ASC LIMIT 11 OFFSET 0").
					WithArgs(
						pgvector.NewVector([]float32{0.1, 0.2, 0.3}),
						pgvector.NewVector([]float32{0.1, 0.2, 0.3}),
					).
					WillReturnRows(rows)
			},
			expectedTodos: []domain.Todo{
				{ID: fixedUUID2, Title: "Todo 2", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
				{ID: fixedUUID1, Title: "Todo 1", Status: domain.TodoStatus_OPEN, DueDate: fixedDueDate, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: false,
			expectedErr:     false,
		},
		"no-embedding-for-similarity-sort": {
			page:     1,
			pageSize: 10,
			opts: []domain.ListTodoOptions{
				domain.WithSortBy("similarityAsc"),
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
			opts: []domain.ListTodoOptions{
				domain.WithSortBy("invalidSort"),
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
			got, hasMore, gotErr := repo.ListTodos(context.Background(), tt.page, tt.pageSize, tt.opts...)
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
			gotErr := repo.DeleteTodo(context.Background(), id)

			if tt.err {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInitTodoRepository_Initialize(t *testing.T) {
	i := &InitTodoRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	_, err = depend.Resolve[domain.TodoRepository]()
	assert.NoError(t, err)
}
