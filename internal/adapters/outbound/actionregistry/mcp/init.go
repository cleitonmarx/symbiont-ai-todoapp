package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitMCPActionRegistry registers the MCP-backed assistant action registry.
type InitMCPActionRegistry struct {
	Logger         *log.Logger   `resolve:""`
	HttpClient     *http.Client  `resolve:""`
	Endpoint       string        `config:"MCP_GATEWAY_ENDPOINT"`
	APIKey         string        `config:"MCP_GATEWAY_API_KEY" default:""`
	APIKeyHeader   string        `config:"MCP_GATEWAY_API_KEY_HEADER" default:""`
	RequestTimeout time.Duration `config:"MCP_GATEWAY_REQUEST_TIMEOUT" default:"20s"`
	registry       *MCPRegistry
}

// Initialize creates and registers the MCP-backed action registry.
func (i *InitMCPActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	_, span := telemetry.StartSpan(ctx)
	defer span.End()

	i.registry = NewMCPRegistry(
		Config{
			Endpoint:       i.Endpoint,
			APIKey:         i.APIKey,
			APIKeyHeader:   i.APIKeyHeader,
			RequestTimeout: i.RequestTimeout,
		},
		i.HttpClient,
	)
	if err := i.registry.initializeActions(ctx); err != nil {
		return ctx, fmt.Errorf("failed to initialize mcp actions: %w", err)
	}
	depend.RegisterNamed[assistant.ActionRegistry](i.registry, "mcp")
	return ctx, nil
}

// Close terminates the MCP session and logs any errors encountered during shutdown.
func (i *InitMCPActionRegistry) Close() {
	if i == nil || i.registry == nil {
		return
	}

	if err := i.registry.Close(); err != nil && i.Logger != nil {
		i.Logger.Printf("InitMCPActionRegistry: failed to close MCP registry: %v", err)
	}
}
