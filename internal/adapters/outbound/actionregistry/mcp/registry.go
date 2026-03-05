package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/attribute"
)

var _ domain.AssistantActionRegistry = (*MCPRegistry)(nil)

// MCPRegistry implements domain.AssistantActionRegistry using a remote MCP gateway.
type MCPRegistry struct {
	cfg           Config
	connector     mcpConnector
	session       mcpSession
	actionsByName map[string]domain.AssistantAction
}

// NewMCPRegistry creates an MCP-backed assistant action registry.
func NewMCPRegistry(
	cfg Config,
	httpClient *http.Client,
) *MCPRegistry {
	cfg = cfg.withDefaults()
	if cfg.APIKey != "" {
		httpClient = withAPIKey(httpClient, cfg.APIKeyHeader, cfg.APIKey)
	}
	return &MCPRegistry{
		cfg:           cfg,
		connector:     streamableConnector{endpoint: cfg.Endpoint, httpClient: httpClient},
		actionsByName: map[string]domain.AssistantAction{},
	}
}

// newMCPRegistryWithConnector allows injecting a fake connector in tests.
func newMCPRegistryWithConnector(cfg Config, connector mcpConnector) *MCPRegistry {
	cfg = cfg.withDefaults()
	return &MCPRegistry{
		cfg:           cfg,
		connector:     connector,
		actionsByName: map[string]domain.AssistantAction{},
	}
}

// Execute runs one MCP tool call and returns the result as a tool message.
func (r *MCPRegistry) Execute(ctx context.Context, call domain.AssistantActionCall, _ []domain.AssistantMessage) domain.AssistantMessage {
	spanCtx, span := telemetry.Start(ctx)
	span.SetAttributes(
		attribute.String("mcp.tool_name", call.Name),
	)
	defer span.End()

	_, knownAction := r.actionsByName[call.Name]
	if !knownAction {
		return actionErrorMessage(call.ID, "unknown_action", fmt.Sprintf("action '%s' is not registered", call.Name))
	}

	arguments, err := parseActionCallArguments(call.Input)
	if err != nil {
		return actionErrorMessage(call.ID, "invalid_arguments", err.Error())
	}
	if formattedArguments, found := toolFormatters.FormatArguments(call.Name, arguments); found {
		arguments = formattedArguments
	}

	if r.session == nil {
		return actionErrorMessage(call.ID, "mcp_not_initialized", "mcp registry was not initialized with a live session")
	}

	callCtx, cancel := r.withTimeout(spanCtx)
	defer cancel()

	result, err := r.session.CallTool(callCtx, &mcp.CallToolParams{
		Name:      call.Name,
		Arguments: arguments,
	})
	if err != nil {
		return actionErrorMessage(call.ID, "mcp_call_error", err.Error())
	}

	content := renderCallToolResult(result)
	if formatted, found := toolFormatters.FormatResult(call.Name, content, call); found {
		return formatted
	}

	if result != nil && result.IsError && !strings.Contains(strings.ToLower(content), "error") {
		content = "error: " + strings.TrimSpace(content)
	}
	if strings.TrimSpace(content) == "" {
		content = "ok"
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: common.Ptr(call.ID),
		Content:      content,
	}
}

// GetDefinition returns one action definition by name.
func (r *MCPRegistry) GetDefinition(actionName string) (domain.AssistantActionDefinition, bool) {
	action, found := r.actionsByName[actionName]
	if !found {
		return domain.AssistantActionDefinition{}, false
	}
	return action.Definition(), true
}

// GetRenderer returns one deterministic action result renderer by action name.
func (r *MCPRegistry) GetRenderer(actionName string) (domain.ActionResultRenderer, bool) {
	action, found := r.actionsByName[actionName]
	if !found {
		return nil, false
	}
	return action.Renderer()
}

// StatusMessage returns a status message for one tool name.
func (r *MCPRegistry) StatusMessage(actionName string) string {
	trimmedActionName := strings.TrimSpace(actionName)
	if trimmedActionName == "" {
		return defaultStatusMessage
	}

	action, found := r.actionsByName[trimmedActionName]
	if !found {
		return defaultStatusMessage
	}
	return action.StatusMessage()
}

// withTimeout applies request timeout defaults to MCP network calls.
func (r *MCPRegistry) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if r.cfg.RequestTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, r.cfg.RequestTimeout)
}

// connectSession opens one MCP client session through the configured connector.
func (r *MCPRegistry) connectSession(ctx context.Context) (mcpSession, error) {
	connectCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	return r.connector.Connect(connectCtx)
}

// initializeActions loads tools once, applies YAML overrides, and precomputes embeddings.
func (r *MCPRegistry) initializeActions(ctx context.Context) error {
	session, err := r.connectSession(ctx)
	if err != nil {
		return err
	}

	overrides, err := loadToolOverrides()
	if err != nil {
		return fmt.Errorf("failed to load mcp tool definition overrides: %w", err)
	}

	listCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	tools, err := listAllTools(listCtx, session)
	if err != nil {
		return err
	}

	actions := make(map[string]domain.AssistantAction, len(tools))
	for _, tool := range tools {
		def := toolToActionDefinition(tool)
		if strings.TrimSpace(def.Name) == "" {
			continue
		}
		statusMessage := ""
		if override, found := overrides.Definitions[def.Name]; found {
			def = mergeAssistantActionDefinition(def, override)
		}
		if overrideStatusMessage, found := overrides.StatusMessages[def.Name]; found {
			statusMessage = overrideStatusMessage
		}

		actions[def.Name] = mcpToolAction{definition: def, statusMessage: statusMessage, execute: r.Execute}
	}

	r.session = session
	r.actionsByName = actions
	return nil
}

// Close terminates the MCP session and releases resources.
func (r *MCPRegistry) Close() error {
	if r.session != nil {
		return r.session.Close()
	}
	return nil
}

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

// Initialize registers this implementation as domain.AssistantActionRegistry.
func (i InitMCPActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	_, span := telemetry.Start(ctx)
	defer span.End()

	registry := NewMCPRegistry(
		Config{
			Endpoint:       i.Endpoint,
			APIKey:         i.APIKey,
			APIKeyHeader:   i.APIKeyHeader,
			RequestTimeout: i.RequestTimeout,
		},
		i.HttpClient,
	)
	if err := registry.initializeActions(ctx); err != nil {
		return ctx, fmt.Errorf("failed to initialize mcp actions: %w", err)
	}
	depend.RegisterNamed[domain.AssistantActionRegistry](registry, "mcp")
	return ctx, nil
}

// Close terminates the MCP session and logs any errors encountered during shutdown.
func (i InitMCPActionRegistry) Close() {
	if i.registry == nil {
		return
	}
	if err := i.registry.Close(); err != nil {
		i.Logger.Printf("InitMCPActionRegistry: failed to close MCP registry: %v", err)
	}
}
