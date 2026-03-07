package mcp

import (
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitMCPActionRegistry_Initialize(t *testing.T) {
	t.Parallel()

	i := InitMCPActionRegistry{
		Logger: log.New(io.Discard, "", 0),
		HttpClient: &http.Client{
			Timeout: 50 * time.Millisecond,
		},
		Endpoint:       "http://127.0.0.1:1",
		RequestTimeout: 50 * time.Millisecond,
	}

	ctx, err := i.Initialize(t.Context())
	require.Error(t, err)
	assert.NotNil(t, ctx)
	assert.Contains(t, err.Error(), "failed to initialize mcp actions")
}

func TestInitMCPActionRegistry_Close(t *testing.T) {
	t.Parallel()

	session := &fakeSession{}
	i := InitMCPActionRegistry{
		Logger:   log.New(io.Discard, "", 0),
		registry: &MCPRegistry{session: session},
	}

	i.Close()
	assert.Equal(t, 1, session.closeCalls)
}

func TestInitMCPActionRegistry_Close_NoRegistry(t *testing.T) {
	t.Parallel()

	i := InitMCPActionRegistry{
		Logger: log.New(io.Discard, "", 0),
	}

	assert.NotPanics(t, func() {
		i.Close()
	})
}
