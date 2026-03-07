package modelrunner

import (
	"context"
	"errors"

	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// AssistantClient adapts DRMAPIClient to domain assistant/model interfaces.
type AssistantClient struct {
	client DRMAPIClient
}

// NewAssistantClientAdapter creates a new adapter.
func NewAssistantClientAdapter(client DRMAPIClient) AssistantClient {
	return AssistantClient{client: client}
}

// RunTurn implements assistant.Assistant.RunTurn.
func (a AssistantClient) RunTurn(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	adapterReq := toChatRequest(req)

	meta := assistant.TurnStarted{
		UserMessageID:      uuid.New(),
		AssistantMessageID: uuid.New(),
	}
	if err := onEvent(spanCtx, assistant.EventType_TurnStarted, meta); err != nil {
		return err
	}

	var (
		actionCalls []*assistant.ActionCall
		usage       assistant.Usage
	)

	err := a.client.ChatStream(spanCtx, adapterReq, func(chunk StreamChunk) error {
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if err := onEvent(spanCtx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: choice.Delta.Content}); err != nil {
					return err
				}
			}
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					if tc.ID != "" {
						actionCalls = append(actionCalls, &assistant.ActionCall{
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
		if err := onEvent(spanCtx, assistant.EventType_ActionRequested, *call); err != nil {
			return err
		}
	}

	return onEvent(spanCtx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
		AssistantMessageID: meta.AssistantMessageID.String(),
		CompletedAt:        time.Now().UTC().Format(time.RFC3339),
		Usage:              usage,
	})
}

// RunTurnSync implements assistant.Assistant.RunTurnSync.
func (a AssistantClient) RunTurnSync(ctx context.Context, req assistant.TurnRequest) (assistant.TurnResponse, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	adapterReq := toChatRequest(req)
	resp, err := a.client.Chat(spanCtx, adapterReq)
	if telemetry.IsErrorRecorded(span, err) {
		return assistant.TurnResponse{}, err
	}
	if len(resp.Choices) == 0 {
		err := errors.New("no choices in response")
		telemetry.IsErrorRecorded(span, err)
		return assistant.TurnResponse{}, err
	}

	res := assistant.TurnResponse{Content: resp.Choices[0].Message.Content}
	if resp.Usage != nil {
		res.Usage = assistant.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	return res, nil
}

// ListAvailableModels returns all available models in a provider-agnostic shape.
func (a AssistantClient) ListAvailableModels(ctx context.Context) ([]assistant.ModelInfo, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	resp, err := a.client.AvailableModels(spanCtx)
	if telemetry.IsErrorRecorded(span, err) {
		return nil, err
	}

	models := make([]assistant.ModelInfo, len(resp.Data))
	for i, m := range resp.Data {
		kind := assistant.ModelKindAssistant
		if strings.Contains(m.ID, "embed") {
			kind = assistant.ModelKindEmbedding
		}
		nameParts := strings.Split(m.ID, "/")
		name := nameParts[len(nameParts)-1]
		models[i] = assistant.ModelInfo{
			ID:   m.ID,
			Name: name,
			Kind: kind,
		}
	}
	return models, nil
}

// ListModels implements assistant.ModelCatalog.ListModels.
func (a AssistantClient) ListModels(ctx context.Context) ([]assistant.ModelCapabilities, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	resp, err := a.client.AvailableModels(spanCtx)
	if telemetry.IsErrorRecorded(span, err) {
		return nil, err
	}

	res := make([]assistant.ModelCapabilities, 0, len(resp.Data))
	for _, m := range resp.Data {
		if strings.Contains(m.ID, "embed") {
			continue
		}
		nameParts := strings.Split(m.ID, "/")
		name := nameParts[len(nameParts)-1]
		res = append(res, assistant.ModelCapabilities{
			ID:                m.ID,
			Name:              name,
			SupportsStreaming: true,
			SupportsActions:   true,
		})
	}
	return res, nil
}

// toChatRequest converts a assistant.TurnRequest to a ChatRequest for the API client.
func toChatRequest(req assistant.TurnRequest) ChatRequest {
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

// mapActionFieldToSchema recursively maps assistant.ActionField to ToolFuncParameterDetail,
// handling nested fields for object types.
func mapActionFieldToSchema(field assistant.ActionField) ToolFuncParameterDetail {
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
