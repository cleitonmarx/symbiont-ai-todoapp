package telemetry

import (
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitOpenTelemetry_Initialize_Close(t *testing.T) {
	t.Parallel()

	init := &InitOpenTelemetry{Logger: log.New(&strings.Builder{}, "", 0)}
	ctx := t.Context()
	ctx, err := init.Initialize(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	init.Close()
}

func TestInitHttpClient_Initialize(t *testing.T) {
	t.Parallel()

	init := InitHttpClient{Logger: log.New(&strings.Builder{}, "", 0)}
	ctx := t.Context()
	ctx, err := init.Initialize(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	standardClient, err := depend.ResolveNamed[*http.Client]("standard")
	assert.NoError(t, err)
	assert.NotNil(t, standardClient)
	assert.Equal(t, defaultHTTPClientTimeout, standardClient.Timeout)

	retryTransport, ok := standardClient.Transport.(*retryablehttp.RoundTripper)
	require.True(t, ok)
	require.NotNil(t, retryTransport.Client)
	assert.Equal(t, 3, retryTransport.Client.RetryMax)
	assert.Equal(t, defaultHTTPClientTimeout, retryTransport.Client.HTTPClient.Timeout)
	assert.NotNil(t, retryTransport.Client.HTTPClient.Transport)

	streamingClient, err := depend.ResolveNamed[*http.Client]("streaming")
	assert.NoError(t, err)
	assert.NotNil(t, streamingClient)
	assert.NotSame(t, standardClient, streamingClient)
	assert.Zero(t, streamingClient.Timeout)

	streamingRetryTransport, ok := streamingClient.Transport.(*retryablehttp.RoundTripper)
	require.True(t, ok)
	require.NotNil(t, streamingRetryTransport.Client)
	assert.Zero(t, streamingRetryTransport.Client.RetryMax)
	assert.Zero(t, streamingRetryTransport.Client.HTTPClient.Timeout)
	assert.NotNil(t, streamingRetryTransport.Client.HTTPClient.Transport)
}
