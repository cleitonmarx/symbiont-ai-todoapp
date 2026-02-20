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
	embeddingFactory EmbeddingFactory
}

// NewAssistantClientAdapter creates a new adapter.
func NewAssistantClientAdapter(client DRMAPIClient) AssistantClient {
	return AssistantClient{client: client, embeddingFactory: embeddingFactory{}}
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
	if err := onEvent(domain.AssistantEventType_TurnStarted, meta); err != nil {
		return err
	}

	var (
		actionCalls []*domain.AssistantActionCall
		usage       domain.AssistantUsage
	)

	err := a.client.ChatStream(spanCtx, adapterReq, func(chunk StreamChunk) error {
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if err := onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: choice.Delta.Content}); err != nil {
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
		if err := onEvent(domain.AssistantEventType_ActionRequested, *call); err != nil {
			return err
		}
	}

	return onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
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

	prompt := a.embeddingFactory.Get(model).GenerateIndexingPrompt(todo)
	vec, err := a.embed(spanCtx, model, prompt)
	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.EmbeddingVector{}, err
	}
	return vec, nil
}

// VectorizeQuery implements domain.SemanticEncoder.
func (a AssistantClient) VectorizeQuery(ctx context.Context, model, query string) (domain.EmbeddingVector, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	prompt := a.embeddingFactory.Get(model).GenerateSearchPrompt(query)
	vec, err := a.embed(spanCtx, model, prompt)
	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.EmbeddingVector{}, err
	}
	return vec, nil
}

// VectorizeAssistantActionDefinition implements domain.SemanticEncoder.
func (a AssistantClient) VectorizeAssistantActionDefinition(ctx context.Context, model string, action domain.AssistantActionDefinition) (domain.EmbeddingVector, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	prompt := a.embeddingFactory.Get(model).GenerateAssistentActionDefinitionPrompt(action)
	vec, err := a.embed(spanCtx, model, prompt)
	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.EmbeddingVector{}, err
	}
	return vec, nil
}

func (a AssistantClient) embed(ctx context.Context, model, input string) (domain.EmbeddingVector, error) {
	req := EmbeddingsRequest{Model: model, Input: input}
	resp, err := a.client.Embeddings(ctx, req)
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
			tool.Function.Parameters.Properties[paramName] = ToolFuncParameterDetail{
				Type:        field.Type,
				Description: field.Description,
			}
			if field.Required {
				tool.Function.Parameters.Required = append(tool.Function.Parameters.Required, paramName)
			}
		}
		adapterReq.Tools[i] = tool
	}

	return adapterReq
}

// InitAssistantClient initializes the assistant client dependency.
type InitAssistantClient struct {
	HttpClient *http.Client `resolve:""`
	ModelHost  string       `config:"LLM_MODEL_HOST"`
}

// Initialize registers assistant/model interfaces.
func (i InitAssistantClient) Initialize(ctx context.Context) (context.Context, error) {
	adapter := NewAssistantClientAdapter(NewDRMAPIClient(i.ModelHost, "", i.HttpClient))
	depend.Register[domain.Assistant](adapter)
	depend.Register[domain.SemanticEncoder](adapter)
	depend.Register[domain.AssistantModelCatalog](adapter)
	return ctx, nil
}
