package mcp

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultRequestTimeout = 20 * time.Second
	defaultStatusMessage  = "⏳ Running MCP tool..."
)

// Config configures the MCP gateway-backed assistant action registry.
type Config struct {
	Endpoint       string
	APIKey         string
	APIKeyHeader   string
	RequestTimeout time.Duration
}

// withDefaults applies safe defaults for header and timeouts.
func (c Config) withDefaults() Config {
	cfg := c
	apiKeyHeader := strings.TrimSpace(cfg.APIKeyHeader)
	if apiKeyHeader == "" {
		cfg.APIKeyHeader = "Authorization"
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	return cfg
}

type mcpSession interface {
	ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
	Close() error
}

type mcpConnector interface {
	Connect(ctx context.Context) (mcpSession, error)
}

type streamableConnector struct {
	endpoint   string
	httpClient *http.Client
}
