package mcp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Connect builds an SDK client and opens a streamable-http MCP session.
func (c streamableConnector) Connect(ctx context.Context) (mcpSession, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	if strings.TrimSpace(c.endpoint) == "" {
		return nil, errors.New("mcp endpoint is empty")
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "todoapp-mcp-client", Version: "v1.0.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:   c.endpoint,
		HTTPClient: c.httpClient,
	}
	return client.Connect(spanCtx, transport, nil)
}

// withAPIKey injects one header into every request by wrapping the provided HTTP transport.
func withAPIKey(httpClient *http.Client, headerName, apiKey string) *http.Client {
	if strings.TrimSpace(apiKey) == "" {
		if httpClient != nil {
			return httpClient
		}
		return &http.Client{}
	}

	base := httpClient
	if base == nil {
		base = &http.Client{}
	}

	clone := *base
	transport := clone.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clone.Transport = authRoundTripper{
		base:       transport,
		headerName: strings.TrimSpace(headerName),
		headerVal:  strings.TrimSpace(apiKey),
	}
	return &clone
}

// authRoundTripper is an HTTP transport wrapper that injects a static header for authentication purposes.
type authRoundTripper struct {
	base       http.RoundTripper
	headerName string
	headerVal  string
}

// RoundTrip clones the request and injects the configured auth header.
func (t authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header.Set(t.headerName, t.headerVal)
	return t.base.RoundTrip(cloned)
}
