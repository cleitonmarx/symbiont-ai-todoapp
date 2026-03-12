package postgres

import (
	"io"
	"log"
	"testing"

	"github.com/XSAM/otelsql"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func Test_withQueryAttributes(t *testing.T) {
	t.Parallel()

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
			got := withQueryAttributes(log.New(io.Discard, "", 0))(t.Context(), otelsql.MethodConnQuery, tt.query, nil)
			assert.Equal(t, tt.want, got)

		})
	}
}
