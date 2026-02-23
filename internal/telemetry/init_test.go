package telemetry

import (
	"context"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitOpenTelemetry_Initialize_Close(t *testing.T) {
	t.Parallel()

	init := &InitOpenTelemetry{Logger: log.New(&strings.Builder{}, "", 0)}
	ctx := context.Background()
	ctx, err := init.Initialize(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	init.Close()
}

func TestInitHttpClient_Initialize(t *testing.T) {
	t.Parallel()

	init := InitHttpClient{Logger: log.New(&strings.Builder{}, "", 0)}
	ctx := context.Background()
	ctx, err := init.Initialize(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
}
