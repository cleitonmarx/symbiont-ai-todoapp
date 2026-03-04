package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_WithDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input Config
		want  Config
	}{
		"fills-header-and-timeout": {
			input: Config{Endpoint: "http://localhost:8811/mcp"},
			want:  Config{Endpoint: "http://localhost:8811/mcp", APIKeyHeader: "Authorization", RequestTimeout: defaultRequestTimeout},
		},
		"keeps-explicit-values": {
			input: Config{Endpoint: "http://localhost:8811/mcp", APIKeyHeader: "X-API-Key", RequestTimeout: 5 * time.Second},
			want:  Config{Endpoint: "http://localhost:8811/mcp", APIKeyHeader: "X-API-Key", RequestTimeout: 5 * time.Second},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.input.withDefaults())
		})
	}
}
