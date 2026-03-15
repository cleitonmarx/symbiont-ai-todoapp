package mcp

import (
	"context"
	"fmt"

	"net/http"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/attribute"
)

var _ assistant.ActionRegistry = (*ActionRegistry)(nil)

// ActionRegistry implements assistant.ActionRegistry using a MCP gateway.
type ActionRegistry struct {
	cfg           Config
	connector     mcpConnector
	session       mcpSession
	actionsByName map[string]assistant.Action
}

// NewActionRegistry creates an MCP-backed assistant action registry.
func NewActionRegistry(
	cfg Config,
	httpClient *http.Client,
) *ActionRegistry {
	cfg = cfg.withDefaults()
	if cfg.APIKey != "" {
		httpClient = withAPIKey(httpClient, cfg.APIKeyHeader, cfg.APIKey)
	}
	return &ActionRegistry{
		cfg:           cfg,
		connector:     streamableConnector{endpoint: cfg.Endpoint, httpClient: httpClient},
		actionsByName: map[string]assistant.Action{},
	}
}

// newActionRegistryWithConnector allows injecting a fake connector in tests.
func newActionRegistryWithConnector(cfg Config, connector mcpConnector) *ActionRegistry {
	cfg = cfg.withDefaults()
	return &ActionRegistry{
		cfg:           cfg,
		connector:     connector,
		actionsByName: map[string]assistant.Action{},
	}
}

// Execute runs one MCP tool call and returns the result as a tool message.
func (r *ActionRegistry) Execute(ctx context.Context, call assistant.ActionCall, _ []assistant.Message) assistant.Message {
	spanCtx, span := telemetry.StartSpan(ctx)
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

	var actionError *string
	if result != nil && result.IsError {
		errText := strings.TrimSpace(content)
		actionError = &errText
	}

	return assistant.Message{
		Role:         assistant.ChatRole_Tool,
		ActionCallID: common.Ptr(call.ID),
		Content:      content,
		ActionError:  actionError,
	}
}

// GetDefinition returns one action definition by name.
func (r *ActionRegistry) GetDefinition(actionName string) (assistant.ActionDefinition, bool) {
	action, found := r.actionsByName[actionName]
	if !found {
		return assistant.ActionDefinition{}, false
	}
	return action.Definition(), true
}

// GetRenderer returns one deterministic action result renderer by action name.
func (r *ActionRegistry) GetRenderer(actionName string) (assistant.ActionResultRenderer, bool) {
	action, found := r.actionsByName[actionName]
	if !found {
		return nil, false
	}
	return action.Renderer()
}

// StatusMessage returns a status message for one tool name.
func (r *ActionRegistry) StatusMessage(actionName string) string {
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
func (r *ActionRegistry) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if r.cfg.RequestTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, r.cfg.RequestTimeout)
}

// connectSession opens one MCP client session through the configured connector.
func (r *ActionRegistry) connectSession(ctx context.Context) (mcpSession, error) {
	connectCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	return r.connector.Connect(connectCtx)
}

// initializeActions loads tools once, applies YAML overrides, and precomputes embeddings.
func (r *ActionRegistry) initializeActions(ctx context.Context) error {
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

	actions := make(map[string]assistant.Action, len(tools))
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
func (r *ActionRegistry) Close() error {
	if r.session != nil {
		return r.session.Close()
	}
	return nil
}
