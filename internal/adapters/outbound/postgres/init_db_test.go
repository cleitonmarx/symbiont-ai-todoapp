package postgres

import (
	"context"
	"database/sql"
	"io"
	"log"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/XSAM/otelsql"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

func TestInitDB_Initialize(t *testing.T) {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	dbInit := InitDB{
		Logger:        logger,
		DBUser:        "testuser",
		DBPass:        "testpass",
		DBHost:        "localhost",
		DBPort:        "5432",
		DBName:        "testdb",
		skipMigration: true,
	}

	_, err := dbInit.Initialize(context.Background())
	assert.NoError(t, err)
	resolveDB, err := depend.Resolve[*sql.DB]()
	assert.NoError(t, err)
	assert.NotNil(t, resolveDB)

}

func TestInitDB_Close(t *testing.T) {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		dbInit          *InitDB
		shouldClose     bool
	}{
		"close-success": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose()
			},
			dbInit: &InitDB{
				Logger: logger,
			},
			shouldClose: true,
		},
		"close-log-error": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose().WillReturnError(sql.ErrConnDone)
			},
			dbInit: &InitDB{
				Logger: logger,
			},
			shouldClose: true,
		},
		"close-with-nil-db": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				// No expectations for nil db
			},
			dbInit: &InitDB{
				Logger: logger,
				db:     nil,
			},
			shouldClose: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.shouldClose {
				db, mock, err := sqlmock.New()
				assert.NoError(t, err)

				tt.setExpectations(mock)
				tt.dbInit.db = db

				tt.dbInit.Close()
				assert.NoError(t, mock.ExpectationsWereMet())
			} else {
				tt.dbInit.Close()
				assert.Nil(t, tt.dbInit.db)
			}
		})
	}
}

func Test_withQueryAttributes(t *testing.T) {
	tests := map[string]struct {
		query string
		want  []attribute.KeyValue
	}{
		"select-query": {
			query: "SELECT * FROM todos WHERE status = 'OPEN'",
			want: []attribute.KeyValue{
				semconv.DBQuerySummary("SELECT todos"),
				semconv.DBCollectionName("todos"),
			},
		},
		"insert-query": {
			query: "INSERT INTO todos (id, title, status) VALUES ($1, $2, $3)",
			want: []attribute.KeyValue{
				semconv.DBQuerySummary("INSERT todos"),
				semconv.DBCollectionName("todos"),
			},
		},
		"update-query": {
			query: "UPDATE todos SET status = 'DONE' WHERE id = $1",
			want: []attribute.KeyValue{
				semconv.DBQuerySummary("UPDATE todos"),
				semconv.DBCollectionName("todos"),
			},
		},
		"delete-query": {
			query: "DELETE FROM todos WHERE id = $1",
			want: []attribute.KeyValue{
				semconv.DBQuerySummary("DELETE todos"),
				semconv.DBCollectionName("todos"),
			},
		},
		"complex-query": {
			query: "WITH recent AS (SELECT * FROM todos ORDER BY created_at DESC LIMIT 10) SELECT * FROM recent WHERE status = 'OPEN'",
			want: []attribute.KeyValue{
				semconv.DBQuerySummary("SELECT todos"),
				semconv.DBCollectionName("todos"),
			},
		},
		"no-table-query": {
			query: "BEGIN TRANSACTION",
			want: []attribute.KeyValue{
				semconv.DBQuerySummary("BEGIN "),
			},
		},
		"malformed-query": {
			query: "SELECT->FROM->WHEd434RE",
			want:  []attribute.KeyValue{},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := withQueryAttributes(log.New(io.Discard, "", 0))(context.Background(), otelsql.MethodConnQuery, tt.query, nil)
			assert.Equal(t, tt.want, got)

		})
	}
}
