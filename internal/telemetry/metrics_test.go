package telemetry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestWithHTTPMetricAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pattern    string
		wantRoute  string
		wantLength int
	}{
		{
			name:       "path pattern only",
			pattern:    "/todos/{id}",
			wantRoute:  "/todos/{id}",
			wantLength: 1,
		},
		{
			name:       "method and path pattern",
			pattern:    "GET /todos/{id}",
			wantRoute:  "/todos/{id}",
			wantLength: 1,
		},
		{
			name:       "host and path pattern",
			pattern:    "example.com/todos/{id}",
			wantRoute:  "/todos/{id}",
			wantLength: 1,
		},
		{
			name:       "no pattern",
			pattern:    "",
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptestRequest(t, tt.pattern)
			attrs := WithHttpMetricAttributes(req)

			assert.Len(t, attrs, tt.wantLength)
			if tt.wantLength == 0 {
				return
			}

			assert.Equal(t, attribute.Key("http.route"), attrs[0].Key)
			assert.Equal(t, tt.wantRoute, attrs[0].Value.AsString())
		})
	}
}

func httptestRequest(t *testing.T, pattern string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, "http://localhost/todos/123", nil)
	if err != nil {
		t.Fatalf("failed creating request: %v", err)
	}
	req.Pattern = pattern
	return req
}
