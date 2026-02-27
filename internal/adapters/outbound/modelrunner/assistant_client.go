package modelrunner

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
)

// AssistantClient adapts DRMAPIClient to domain assistant/model interfaces.
type AssistantClient struct {
	client           DRMAPIClient
	embeddingClient  DRMAPIClient
	embeddingFactory EmbeddingFactory
}

// NewAssistantClientAdapter creates a new adapter.
func NewAssistantClientAdapter(client DRMAPIClient, embeddingClient DRMAPIClient) AssistantClient {
	return AssistantClient{client: client, embeddingClient: embeddingClient, embeddingFactory: embeddingFactory{}}
}

// RunTurn implements domain.Assistant.
func (a AssistantClient) RunTurn(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	adapterReq := toChatRequest(req)

	meta := domain.AssistantTurnStarted{
		UserMessageID:      uuid.New(),
		AssistantMessageID: uuid.New(),
	}
	if err := onEvent(spanCtx, domain.AssistantEventType_TurnStarted, meta); err != nil {
		return err
	}

	var (
		actionCalls []*domain.AssistantActionCall
		usage       domain.AssistantUsage
	)

	err := a.client.ChatStream(spanCtx, adapterReq, func(chunk StreamChunk) error {
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if err := onEvent(spanCtx, domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: choice.Delta.Content}); err != nil {
					return err
				}
			}
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					if tc.ID != "" {
						actionCalls = append(actionCalls, &domain.AssistantActionCall{
							ID:    tc.ID,
							Name:  tc.Function.Name,
							Input: tc.Function.Arguments,
						})
						continue
					}
					if tc.Index >= 0 && tc.Index < len(actionCalls) {
						actionCalls[tc.Index].Input += tc.Function.Arguments
					}
				}
			}
		}

		if chunk.Usage != nil {
			usage.PromptTokens = chunk.Usage.PromptTokens
			usage.CompletionTokens = chunk.Usage.CompletionTokens
			usage.TotalTokens = chunk.Usage.TotalTokens
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, call := range actionCalls {
		if err := onEvent(spanCtx, domain.AssistantEventType_ActionRequested, *call); err != nil {
			return err
		}
	}

	return onEvent(spanCtx, domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
		AssistantMessageID: meta.AssistantMessageID.String(),
		CompletedAt:        time.Now().UTC().Format(time.RFC3339),
		Usage:              usage,
	})
}

// RunTurnSync implements domain.Assistant.
func (a AssistantClient) RunTurnSync(ctx context.Context, req domain.AssistantTurnRequest) (domain.AssistantTurnResponse, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	adapterReq := toChatRequest(req)
	resp, err := a.client.Chat(spanCtx, adapterReq)
	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.AssistantTurnResponse{}, err
	}
	if len(resp.Choices) == 0 {
		err := errors.New("no choices in response")
		telemetry.RecordErrorAndStatus(span, err)
		return domain.AssistantTurnResponse{}, err
	}

	res := domain.AssistantTurnResponse{Content: resp.Choices[0].Message.Content}
	if resp.Usage != nil {
		res.Usage = domain.AssistantUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	return res, nil
}

// VectorizeTodo implements domain.SemanticEncoder.
func (a AssistantClient) VectorizeTodo(ctx context.Context, model string, todo domain.Todo) (domain.EmbeddingVector, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()
	gen := a.embeddingFactory.Get(model)
	prompt := gen.GenerateIndexingPrompt(todo.Title)
	dimension := gen.Dimensions()
	vec, err := a.embed(spanCtx, model, prompt, dimension)
	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.EmbeddingVector{}, err
	}
	return vec, nil
}

// VectorizeQuery implements domain.SemanticEncoder.
func (a AssistantClient) VectorizeQuery(ctx context.Context, model, query string) (domain.EmbeddingVector, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	gen := a.embeddingFactory.Get(model)
	prompt := gen.GenerateSearchPrompt(query)
	dimension := gen.Dimensions()
	vec, err := a.embed(spanCtx, model, prompt, dimension)
	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.EmbeddingVector{}, err
	}
	return vec, nil
}

// VectorizeSkillDefinition implements domain.SemanticEncoder.
func (a AssistantClient) VectorizeSkillDefinition(
	ctx context.Context,
	model string,
	skill domain.AssistantSkillDefinition,
) (domain.EmbeddingVector, domain.EmbeddingVector, error) {
	gen := a.embeddingFactory.Get(model)
	dimension := gen.Dimensions()
	var (
		useVector domain.EmbeddingVector
		err       error
	)
	if strings.TrimSpace(skill.UseWhen) != "" {
		useText := gen.GenerateIndexingPrompt(buildSkillUseEmbeddingText(skill))
		useVector, err = a.embed(ctx, model, useText, dimension)
		if err != nil {
			return domain.EmbeddingVector{}, domain.EmbeddingVector{}, err
		}
	}

	var avoidVector domain.EmbeddingVector
	if strings.TrimSpace(skill.AvoidWhen) != "" {
		avoidText := gen.GenerateIndexingPrompt(buildSkillAvoidEmbeddingText(skill))
		avoidVector, err = a.embed(ctx, model, avoidText, dimension)
		if err != nil {
			return domain.EmbeddingVector{}, domain.EmbeddingVector{}, err
		}
	}
	return useVector, avoidVector, nil
}

func buildSkillUseEmbeddingText(skill domain.AssistantSkillDefinition) string {
	parts := make([]string, 0, 5)
	parts = appendIfNotEmpty(parts, "name: "+strings.TrimSpace(skill.Name))
	parts = appendIfNotEmpty(parts, "use_when: "+strings.TrimSpace(skill.UseWhen))
	if len(skill.Tags) > 0 {
		parts = append(parts, "tags: "+strings.Join(skill.Tags, ", "))
	}
	if len(skill.Tools) > 0 {
		parts = append(parts, "tools: "+strings.Join(skill.Tools, ", "))
	}
	return strings.Join(parts, "\n")
}

func buildSkillAvoidEmbeddingText(skill domain.AssistantSkillDefinition) string {
	avoid := strings.TrimSpace(skill.AvoidWhen)
	if avoid == "" {
		return ""
	}
	return "avoid_when: " + avoid
}

func appendIfNotEmpty(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	return append(values, value)
}

func (a AssistantClient) embed(ctx context.Context, model, input string, dimension *int) (domain.EmbeddingVector, error) {
	req := EmbeddingsRequest{Model: model, Input: input, Dimensions: dimension}
	resp, err := a.embeddingClient.Embeddings(ctx, req)
	if err != nil {
		return domain.EmbeddingVector{}, err
	}
	if len(resp.Data) == 0 {
		return domain.EmbeddingVector{}, errors.New("no embedding data in response")
	}
	return domain.EmbeddingVector{
		Vector:      resp.Data[0].Embedding,
		TotalTokens: resp.Usage.TotalTokens,
	}, nil
}

// ListAvailableModels returns all available models in a provider-agnostic shape.
func (a AssistantClient) ListAvailableModels(ctx context.Context) ([]domain.ModelInfo, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	resp, err := a.client.AvailableModels(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, err
	}

	models := make([]domain.ModelInfo, len(resp.Data))
	for i, m := range resp.Data {
		kind := domain.ModelKindAssistant
		if strings.Contains(m.ID, "embed") {
			kind = domain.ModelKindEmbedding
		}
		models[i] = domain.ModelInfo{
			Name: strings.TrimPrefix(m.ID, "docker.io/ai/"),
			Kind: kind,
		}
	}
	return models, nil
}

// ListAssistantModels implements domain.AssistantModelCatalog.
func (a AssistantClient) ListAssistantModels(ctx context.Context) ([]domain.AssistantModelInfo, error) {
	models, err := a.ListAvailableModels(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]domain.AssistantModelInfo, 0, len(models))
	for _, m := range models {
		if m.Kind != domain.ModelKindAssistant {
			continue
		}
		res = append(res, domain.AssistantModelInfo{
			Name:              m.Name,
			SupportsStreaming: true,
			SupportsActions:   true,
		})
	}
	return res, nil
}

// toChatRequest converts a domain.AssistantTurnRequest to a ChatRequest for the API client.
func toChatRequest(req domain.AssistantTurnRequest) ChatRequest {
	adapterReq := ChatRequest{
		Model:            req.Model,
		Temperature:      req.Temperature,
		Stream:           req.Stream,
		MaxTokens:        req.MaxTokens,
		TopP:             req.TopP,
		FrequencyPenalty: req.FrequencyPenalty,
		Messages:         make([]ChatMessage, len(req.Messages)),
		Tools:            make([]Tool, len(req.AvailableActions)),
	}

	if req.Stream {
		adapterReq.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	for i, msg := range req.Messages {
		adpMsg := ChatMessage{
			Role:       string(msg.Role),
			ToolCallID: msg.ActionCallID,
			Content:    msg.Content,
		}
		for _, actionCall := range msg.ActionCalls {
			adpMsg.ToolCalls = append(adpMsg.ToolCalls, ToolCall{
				ID:   actionCall.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      actionCall.Name,
					Arguments: actionCall.Input,
				},
			})
		}
		adapterReq.Messages[i] = adpMsg
	}

	for i, action := range req.AvailableActions {
		tool := Tool{
			Type: "function",
			Function: ToolFunc{
				Description: action.Description,
				Name:        action.Name,
				Parameters: ToolFuncParameters{
					Type:       action.Input.Type,
					Properties: make(map[string]ToolFuncParameterDetail),
					Required:   []string{},
				},
			},
		}

		for paramName, field := range action.Input.Fields {
			tool.Function.Parameters.Properties[paramName] = mapActionFieldToSchema(field)
			if field.Required {
				tool.Function.Parameters.Required = append(tool.Function.Parameters.Required, paramName)
			}
		}
		adapterReq.Tools[i] = tool
	}

	return adapterReq
}

// mapActionFieldToSchema recursively maps domain.AssistantActionField to ToolFuncParameterDetail,
// handling nested fields for object types.
func mapActionFieldToSchema(field domain.AssistantActionField) ToolFuncParameterDetail {
	schema := ToolFuncParameterDetail{
		Type:        field.Type,
		Description: field.Description,
		Format:      field.Format,
		Enum:        field.Enum,
	}
	if field.Type == "object" {
		schema.AdditionalProperties = false
	}

	if len(field.Fields) > 0 {
		schema.Properties = make(map[string]ToolFuncParameterDetail, len(field.Fields))
		required := make([]string, 0, len(field.Fields))
		for name, child := range field.Fields {
			schema.Properties[name] = mapActionFieldToSchema(child)
			if child.Required {
				required = append(required, name)
			}
		}
		if len(required) > 0 {
			schema.Required = required
		}
		schema.AdditionalProperties = false
	}

	if field.Items != nil {
		itemSchema := mapActionFieldToSchema(*field.Items)
		schema.Items = &itemSchema
	} else if field.Type == "array" {
		// safety fallback to avoid invalid schema
		schema.Items = &ToolFuncParameterDetail{Type: "object"}
	}

	return schema
}

// InitAssistantClient initializes the assistant client dependency.
type InitAssistantClient struct {
	HttpClient         *http.Client `resolve:""`
	EmbeddingModelHost string       `config:"LLM_EMBEDDING_MODEL_HOST"`
	ModelHost          string       `config:"LLM_MODEL_HOST"`
	APIKey             string       `config:"LLM_API_KEY" default:"-"`
	EmbeddingAPIKey    string       `config:"LLM_EMBEDDING_API_KEY" default:"-"`
}

// Initialize registers assistant/model interfaces.
func (i InitAssistantClient) Initialize(ctx context.Context) (context.Context, error) {
	apiKey := ""
	if i.APIKey != "-" {
		apiKey = i.APIKey
	}
	embeddingAPIKey := ""
	if i.EmbeddingAPIKey != "-" {
		embeddingAPIKey = i.EmbeddingAPIKey
	}

	adapter := NewAssistantClientAdapter(
		NewDRMAPIClient(i.ModelHost, apiKey, i.HttpClient),
		NewDRMAPIClient(i.EmbeddingModelHost, embeddingAPIKey, i.HttpClient),
	)
	depend.Register[domain.Assistant](adapter)
	depend.Register[domain.SemanticEncoder](adapter)
	depend.Register[domain.AssistantModelCatalog](adapter)
	return ctx, nil
}
