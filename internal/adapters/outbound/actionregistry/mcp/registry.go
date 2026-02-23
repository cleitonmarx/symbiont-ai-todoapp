package mcp

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toon-format/toon-go"
	"go.opentelemetry.io/otel/attribute"
	"go.yaml.in/yaml/v3"
)

const (
	defaultRelevantActionsTopK     = 3
	defaultRelevantActionsMinScore = 0.35
	defaultRequestTimeout          = 20 * time.Second
	defaultStatusMessage           = "⏳ Running MCP tool..."
)

var _ domain.AssistantActionRegistry = (*MCPRegistry)(nil)
var _ actionregistry.EmbeddingActionRegistry = (*MCPRegistry)(nil)

//go:embed tool_overrides.yaml
var toolOverridesFS embed.FS

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
	if apiKeyHeader == "" || apiKeyHeader == "-" {
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

type mcpToolAction struct {
	definition domain.AssistantActionDefinition
	execute    func(context.Context, domain.AssistantActionCall, []domain.AssistantMessage) domain.AssistantMessage
}

// Definition returns the static action definition associated with this MCP tool.
func (a mcpToolAction) Definition() domain.AssistantActionDefinition {
	return a.definition
}

// StatusMessage returns a per-tool execution status string for UI streaming updates.
func (a mcpToolAction) StatusMessage() string {
	name := strings.TrimSpace(a.definition.Name)
	if name == "" {
		return defaultStatusMessage
	}
	return "⏳ Running " + name + "..."
}

// Execute delegates execution to the registry callback bound at initialization time.
func (a mcpToolAction) Execute(ctx context.Context, call domain.AssistantActionCall, history []domain.AssistantMessage) domain.AssistantMessage {
	if a.execute == nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: common.Ptr(call.ID),
			Content:      "errors[1]{error,details}mcp_call_error,action is not executable",
		}
	}
	return a.execute(ctx, call, history)
}

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

// MCPRegistry implements domain.AssistantActionRegistry using a remote MCP gateway.
type MCPRegistry struct {
	cfg             Config
	connector       mcpConnector
	semanticEncoder domain.SemanticEncoder
	embeddingModel  string

	session       mcpSession
	actionsByName map[string]actionregistry.ActionEmbedding
}

// NewMCPRegistry creates an MCP-backed assistant action registry.
func NewMCPRegistry(
	cfg Config,
	httpClient *http.Client,
	semanticEncoder domain.SemanticEncoder,
	embeddingModel string,
) *MCPRegistry {
	cfg = cfg.withDefaults()
	if cfg.APIKey != "-" {
		httpClient = withAPIKey(httpClient, cfg.APIKeyHeader, cfg.APIKey)
	}
	return &MCPRegistry{
		cfg:             cfg,
		connector:       streamableConnector{endpoint: cfg.Endpoint, httpClient: httpClient},
		semanticEncoder: semanticEncoder,
		embeddingModel:  embeddingModel,
		actionsByName:   map[string]actionregistry.ActionEmbedding{},
	}
}

// newMCPRegistryWithConnector allows injecting a fake connector in tests.
func newMCPRegistryWithConnector(cfg Config, connector mcpConnector, semanticEncoder domain.SemanticEncoder, embeddingModel string) *MCPRegistry {
	cfg = cfg.withDefaults()
	return &MCPRegistry{
		cfg:             cfg,
		connector:       connector,
		semanticEncoder: semanticEncoder,
		embeddingModel:  embeddingModel,
		actionsByName:   map[string]actionregistry.ActionEmbedding{},
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

// StatusMessage returns a status message for one tool name.
func (r *MCPRegistry) StatusMessage(actionName string) string {
	if strings.TrimSpace(actionName) == "" {
		return defaultStatusMessage
	}
	return "⏳ Running " + strings.TrimSpace(actionName) + "..."
}

// ListEmbeddings returns all tool definitions currently available from MCP.
func (r *MCPRegistry) ListEmbeddings(ctx context.Context) []actionregistry.ActionEmbedding {
	_, span := telemetry.Start(ctx)
	defer span.End()

	return copySortedEmbeddings(r.actionsByName)
}

// ListRelevant returns semantically relevant tools for user input.
// Falls back to all tools when embeddings are unavailable.
func (r *MCPRegistry) ListRelevant(ctx context.Context, userInput string) []domain.AssistantActionDefinition {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	allActions := r.ListEmbeddings(spanCtx)
	if len(allActions) == 0 {
		return nil
	}

	if r.semanticEncoder == nil || strings.TrimSpace(r.embeddingModel) == "" {
		return definitionsFromEmbeddings(allActions)
	}

	queryCtx, cancel := r.withTimeout(spanCtx)
	defer cancel()

	queryVector, err := r.semanticEncoder.VectorizeQuery(queryCtx, r.embeddingModel, userInput)
	if err != nil || len(queryVector.Vector) == 0 {
		return definitionsFromEmbeddings(allActions)
	}

	type scoredAction struct {
		definition domain.AssistantActionDefinition
		score      float64
	}

	scored := make([]scoredAction, 0, len(allActions))
	for _, action := range allActions {
		if len(action.Embedding) == 0 {
			continue
		}

		def := action.Action.Definition()
		score, ok := common.CosineSimilarity(queryVector.Vector, action.Embedding)
		if !ok || score < defaultRelevantActionsMinScore {
			continue
		}
		scored = append(scored, scoredAction{
			definition: def,
			score:      score,
		})
	}

	if len(scored) == 0 {
		return definitionsFromEmbeddings(allActions)
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].definition.Name < scored[j].definition.Name
		}
		return scored[i].score > scored[j].score
	})

	limit := min(len(scored), defaultRelevantActionsTopK)
	relevant := make([]domain.AssistantActionDefinition, 0, limit)
	for i := range limit {
		relevant = append(relevant, scored[i].definition)
	}
	return relevant
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

	overrideByToolName, err := loadToolDefinitionOverrides()
	if err != nil {
		return fmt.Errorf("failed to load mcp tool definition overrides: %w", err)
	}

	listCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	tools, err := listAllTools(listCtx, session)
	if err != nil {
		return err
	}

	actions := make(map[string]actionregistry.ActionEmbedding, len(tools))
	for _, tool := range tools {
		def := toolToActionDefinition(tool)
		if strings.TrimSpace(def.Name) == "" {
			continue
		}
		if override, found := overrideByToolName[def.Name]; found {
			def = mergeAssistantActionDefinition(def, override)
		}

		var embedding []float64
		if r.semanticEncoder != nil && strings.TrimSpace(r.embeddingModel) != "" {
			vectorCtx, cancel := r.withTimeout(ctx)
			vector, err := r.semanticEncoder.VectorizeAssistantActionDefinition(vectorCtx, r.embeddingModel, def)
			cancel()
			if err == nil {
				embedding = vector.Vector
			}
		}

		actions[def.Name] = actionregistry.ActionEmbedding{
			Action:    mcpToolAction{definition: def, execute: r.Execute},
			Embedding: embedding,
		}
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

// listAllTools paginates through MCP ListTools until no cursor is returned.
func listAllTools(ctx context.Context, session mcpSession) ([]*mcp.Tool, error) {
	tools := make([]*mcp.Tool, 0)
	cursor := ""

	for {
		res, err := session.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		if res == nil {
			return tools, nil
		}

		tools = append(tools, res.Tools...)
		next := strings.TrimSpace(res.NextCursor)
		if next == "" || next == cursor {
			return tools, nil
		}
		cursor = next
	}
}

// toolToActionDefinition converts an MCP tool schema into the domain action format.
func toolToActionDefinition(tool *mcp.Tool) domain.AssistantActionDefinition {
	if tool == nil {
		return domain.AssistantActionDefinition{}
	}

	description := strings.TrimSpace(tool.Description)
	if description == "" {
		description = strings.TrimSpace(tool.Title)
	}

	return domain.AssistantActionDefinition{
		Name:        strings.TrimSpace(tool.Name),
		Description: description,
		Input:       schemaToInput(tool.InputSchema),
	}
}

type toolOverrideConfig struct {
	Tools []assistantActionDefinitionOverride `yaml:"tools"`
}

type assistantActionDefinitionOverride struct {
	Name        string                     `yaml:"name"`
	Description string                     `yaml:"description"`
	Input       assistantActionInputConfig `yaml:"input"`
	Hints       assistantActionHintsConfig `yaml:"hints"`
}

type assistantActionInputConfig struct {
	Type   string                                  `yaml:"type"`
	Fields map[string]assistantActionFieldOverride `yaml:"fields"`
}

type assistantActionFieldOverride struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

type assistantActionHintsConfig struct {
	UseWhen   string `yaml:"use_when"`
	AvoidWhen string `yaml:"avoid_when"`
	ArgRules  string `yaml:"arg_rules"`
}

func loadToolDefinitionOverrides() (map[string]domain.AssistantActionDefinition, error) {
	embeddedBytes, err := toolOverridesFS.ReadFile("tool_overrides.yaml")
	if err != nil {
		return nil, err
	}
	return parseToolOverrideDefinitions(embeddedBytes)
}

// parseToolOverrideDefinitions parses the embedded YAML override file into action definitions.
func parseToolOverrideDefinitions(content []byte) (map[string]domain.AssistantActionDefinition, error) {
	var cfg toolOverrideConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}

	byName := map[string]domain.AssistantActionDefinition{}
	for _, override := range cfg.Tools {
		name := strings.TrimSpace(override.Name)
		if name == "" {
			continue
		}

		fields := map[string]domain.AssistantActionField{}
		for fieldName, field := range override.Input.Fields {
			fields[fieldName] = domain.AssistantActionField{
				Type:        strings.TrimSpace(field.Type),
				Description: strings.TrimSpace(field.Description),
				Required:    field.Required,
			}
		}

		def := domain.AssistantActionDefinition{
			Name:        name,
			Description: strings.TrimSpace(override.Description),
			Input: domain.AssistantActionInput{
				Type:   strings.TrimSpace(override.Input.Type),
				Fields: fields,
			},
			Hints: domain.AssistantActionHints{
				UseWhen:   strings.TrimSpace(override.Hints.UseWhen),
				AvoidWhen: strings.TrimSpace(override.Hints.AvoidWhen),
				ArgRules:  strings.TrimSpace(override.Hints.ArgRules),
			},
		}
		byName[name] = def
	}
	return byName, nil
}

// mergeAssistantActionDefinition overlays configured overrides on top of discovered tool metadata.
func mergeAssistantActionDefinition(base, override domain.AssistantActionDefinition) domain.AssistantActionDefinition {
	merged := base

	if name := strings.TrimSpace(override.Name); name != "" {
		merged.Name = name
	}
	if description := strings.TrimSpace(override.Description); description != "" {
		merged.Description = description
	}

	if inputType := strings.TrimSpace(override.Input.Type); inputType != "" {
		merged.Input.Type = inputType
	}
	if len(merged.Input.Fields) == 0 {
		merged.Input.Fields = map[string]domain.AssistantActionField{}
	}
	maps.Copy(merged.Input.Fields, override.Input.Fields)

	if override.HasHints() {
		merged.Hints = override.Hints
	}
	return merged
}

// schemaToInput extracts a simplified action input definition from JSON Schema-like MCP input.
func schemaToInput(schema any) domain.AssistantActionInput {
	input := domain.AssistantActionInput{
		Type:   "object",
		Fields: map[string]domain.AssistantActionField{},
	}

	schemaMap, ok := anyToMap(schema)
	if !ok || len(schemaMap) == 0 {
		return input
	}

	if schemaType := strings.TrimSpace(asString(schemaMap["type"])); schemaType != "" {
		input.Type = schemaType
	}

	required := requiredSet(schemaMap["required"])
	props, ok := anyToMap(schemaMap["properties"])
	if !ok {
		return input
	}

	for fieldName, fieldSchemaRaw := range props {
		fieldSchema, _ := anyToMap(fieldSchemaRaw)
		input.Fields[fieldName] = domain.AssistantActionField{
			Type:        schemaFieldType(fieldSchema),
			Description: strings.TrimSpace(asString(fieldSchema["description"])),
			Required:    required[fieldName],
		}
	}

	return input
}

// schemaFieldType resolves field type from direct or composed schema nodes (anyOf/oneOf/allOf).
func schemaFieldType(fieldSchema map[string]any) string {
	if len(fieldSchema) == 0 {
		return ""
	}

	if direct := strings.TrimSpace(parseTypeValue(fieldSchema["type"])); direct != "" {
		return direct
	}

	compoundKeys := []string{"anyOf", "oneOf", "allOf"}
	typeCandidates := make([]string, 0, 2)
	for _, key := range compoundKeys {
		values, ok := fieldSchema[key].([]any)
		if !ok {
			continue
		}
		for _, raw := range values {
			inner, ok := anyToMap(raw)
			if !ok {
				continue
			}
			if typ := strings.TrimSpace(parseTypeValue(inner["type"])); typ != "" {
				typeCandidates = append(typeCandidates, typ)
			}
		}
	}

	if len(typeCandidates) == 0 {
		return ""
	}
	typeCandidates = slices.Compact(typeCandidates)
	sort.Strings(typeCandidates)
	return strings.Join(typeCandidates, "|")
}

// parseTypeValue normalizes type declarations that may be a single value or an array of values.
func parseTypeValue(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case []any:
		candidates := make([]string, 0, len(value))
		for _, item := range value {
			itemString := strings.TrimSpace(asString(item))
			if itemString == "" {
				continue
			}
			candidates = append(candidates, itemString)
		}
		if len(candidates) == 0 {
			return ""
		}
		candidates = slices.Compact(candidates)
		sort.Strings(candidates)
		return strings.Join(candidates, "|")
	default:
		return ""
	}
}

// requiredSet converts a JSON schema required array to a lookup set.
func requiredSet(raw any) map[string]bool {
	set := map[string]bool{}
	values, ok := raw.([]any)
	if !ok {
		return set
	}
	for _, value := range values {
		name := strings.TrimSpace(asString(value))
		if name == "" {
			continue
		}
		set[name] = true
	}
	return set
}

// parseActionCallArguments validates assistant tool input and guarantees a JSON object payload.
func parseActionCallArguments(input string) (map[string]any, error) {
	if strings.TrimSpace(input) == "" {
		return map[string]any{}, nil
	}

	decoder := json.NewDecoder(strings.NewReader(input))
	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}

	args, ok := payload.(map[string]any)
	if !ok {
		return nil, errors.New("action arguments must be a JSON object")
	}
	return args, nil
}

// renderCallToolResult flattens MCP call output into plain text for tool messages.
func renderCallToolResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	if result.StructuredContent != nil {
		if bytes, err := toon.Marshal(result.StructuredContent); err == nil {
			return string(bytes)
		}
	}

	parts := make([]string, 0, len(result.Content)+1)
	for _, content := range result.Content {
		text := strings.TrimSpace(renderContent(content))
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n")
	}

	return ""
}

// renderContent converts one MCP content variant to a user-facing string representation.
func renderContent(content mcp.Content) string {
	switch item := content.(type) {
	case *mcp.TextContent:
		return item.Text
	case *mcp.ImageContent:
		return fmt.Sprintf("[image mime_type=%s bytes=%d]", item.MIMEType, len(item.Data))
	case *mcp.AudioContent:
		return fmt.Sprintf("[audio mime_type=%s bytes=%d]", item.MIMEType, len(item.Data))
	case *mcp.ResourceLink:
		return fmt.Sprintf("[resource_link uri=%s name=%s]", item.URI, item.Name)
	case *mcp.EmbeddedResource:
		if item.Resource == nil {
			return "[embedded_resource]"
		}
		if item.Resource.Text != "" {
			return item.Resource.Text
		}
		if len(item.Resource.Blob) > 0 {
			return fmt.Sprintf("[embedded_resource_blob uri=%s bytes=%d]", item.Resource.URI, len(item.Resource.Blob))
		}
		return fmt.Sprintf("[embedded_resource uri=%s]", item.Resource.URI)
	default:
		bytes, err := json.Marshal(item)
		if err != nil {
			return fmt.Sprintf("%v", item)
		}
		return string(bytes)
	}
}

// actionErrorMessage formats a structured tool error payload consumed by the assistant loop.
func actionErrorMessage(callID, code, details string) domain.AssistantMessage {
	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: common.Ptr(callID),
		Content:      fmt.Sprintf("errors[1]{error,details}%s,%s", code, details),
	}
}

// copySortedEmbeddings returns a stable name-ordered snapshot of action embeddings.
func copySortedEmbeddings(actionsByName map[string]actionregistry.ActionEmbedding) []actionregistry.ActionEmbedding {
	actions := make([]actionregistry.ActionEmbedding, 0, len(actionsByName))
	for _, action := range actionsByName {
		actions = append(actions, action)
	}
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Action.Definition().Name < actions[j].Action.Definition().Name
	})
	return actions
}

// definitionsFromEmbeddings strips embedding vectors and returns only action definitions.
func definitionsFromEmbeddings(actions []actionregistry.ActionEmbedding) []domain.AssistantActionDefinition {
	definitions := make([]domain.AssistantActionDefinition, 0, len(actions))
	for _, action := range actions {
		definitions = append(definitions, action.Action.Definition())
	}
	return definitions
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

// anyToMap best-effort normalizes unknown schema values to map[string]any.
func anyToMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case nil:
		return nil, false
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, false
		}
		return out, true
	}
}

// asString stringifies unknown values for permissive schema parsing.
func asString(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		return fmt.Sprintf("%v", v)
	}
}

// InitMCPActionRegistry registers the MCP-backed assistant action registry.
type InitMCPActionRegistry struct {
	Logger                *log.Logger            `resolve:""`
	HttpClient            *http.Client           `resolve:""`
	SemanticEncoder       domain.SemanticEncoder `resolve:""`
	EmbeddingModel        string                 `config:"LLM_EMBEDDING_MODEL"`
	Endpoint              string                 `config:"MCP_GATEWAY_ENDPOINT"`
	APIKey                string                 `config:"MCP_GATEWAY_API_KEY" default:"-"`
	APIKeyHeader          string                 `config:"MCP_GATEWAY_API_KEY_HEADER" default:"-"`
	RequestTimeout        time.Duration          `config:"MCP_GATEWAY_REQUEST_TIMEOUT" default:"20s"`
	TopActionsPerRegistry int                    `config:"MCP_GATEWAY_TOP_ACTIONS_PER_REGISTRY" default:"2"`
	registry              *MCPRegistry
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
		i.SemanticEncoder,
		i.EmbeddingModel,
	)
	if err := registry.initializeActions(ctx); err != nil {
		return ctx, fmt.Errorf("failed to initialize mcp actions: %w", err)
	}
	depend.RegisterNamed[actionregistry.EmbeddingActionRegistry](registry, "mcp")
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
