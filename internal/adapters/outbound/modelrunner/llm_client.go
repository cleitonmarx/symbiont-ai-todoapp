package modelrunner

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
)

// LLMClient adapts DRMAPIClient to domain.LLMClient interface
type LLMClient struct {
	client DRMAPIClient
}

// NewLLMClientAdapter creates a new adapter
func NewLLMClientAdapter(client DRMAPIClient) LLMClient {
	return LLMClient{client: client}
}

// Chat implements domain.LLMClient.Chat
func (a LLMClient) Chat(ctx context.Context, req domain.LLMChatRequest) (string, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	adapterReq := toChatRequest(req)

	for i, msg := range req.Messages {
		adapterReq.Messages[i] = ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	resp, err := a.client.Chat(spanCtx, adapterReq)
	if tracing.RecordErrorAndStatus(span, err) {
		return "", err
	}

	if len(resp.Choices) == 0 {
		err := errors.New("no choices in response")
		tracing.RecordErrorAndStatus(span, err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

// ChatStream implements domain.LLMClient.ChatStream
func (a LLMClient) ChatStream(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	adapterReq := toChatRequest(req)

	// Send meta event
	meta := domain.LLMStreamEventMeta{
		ConversationID:     domain.GlobalConversationID,
		UserMessageID:      uuid.New(),
		AssistantMessageID: uuid.New(),
		StartedAt:          time.Now().UTC(),
	}
	if err := onEvent(domain.LLMStreamEventType_Meta, meta); err != nil {
		return err
	}

	var (
		functionCalls []*domain.LLMStreamEventFunctionCall
	)

	// Stream chunks
	err := a.client.ChatStream(spanCtx, adapterReq, func(chunk StreamChunk) error {

		// Send delta and accumulate function calls
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if err := onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{
					Text: choice.Delta.Content,
				}); err != nil {
					return err
				}
			}
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					if len(tc.ID) > 0 {
						functionCalls = append(functionCalls, &domain.LLMStreamEventFunctionCall{
							ID:        tc.ID,
							Index:     tc.Index,
							Function:  tc.Function.Name,
							Arguments: tc.Function.Arguments,
						})
					} else {
						fCall := functionCalls[tc.Index]
						fCall.Arguments += tc.Function.Arguments
					}

				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Send function call events
	for _, fc := range functionCalls {
		if err := onEvent(domain.LLMStreamEventType_FunctionCall, *fc); err != nil {
			return err
		}
	}

	// Send done event
	done := domain.LLMStreamEventDone{
		AssistantMessageID: meta.AssistantMessageID.String(),
		CompletedAt:        time.Now().UTC().Format(time.RFC3339),
	}
	return onEvent(domain.LLMStreamEventType_Done, done)
}

func (a LLMClient) Embed(ctx context.Context, model, input string) ([]float64, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	req := EmbeddingsRequest{
		Model: model,
		Input: input,
	}

	resp, err := a.client.Embeddings(spanCtx, req)
	if tracing.RecordErrorAndStatus(span, err) {
		return nil, err
	}

	if len(resp.Data) == 0 {
		err := errors.New("no embedding data in response")
		tracing.RecordErrorAndStatus(span, err)
		return nil, err
	}

	return resp.Data[0].Embedding, nil
}

// toChatRequest converts domain.LLMChatRequest to ChatRequest
func toChatRequest(req domain.LLMChatRequest) ChatRequest {
	adapterReq := ChatRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		Stream:      req.Stream,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Messages:    make([]ChatMessage, len(req.Messages)),
		Tools:       make([]Tool, len(req.Tools)),
	}

	for i, msg := range req.Messages {
		adpMsg := ChatMessage{
			Role:       string(msg.Role),
			ToolCallID: msg.ToolCallID,
			Content:    msg.Content,
		}
		for _, fc := range msg.ToolCalls {
			adpMsg.ToolCalls = append(adpMsg.ToolCalls, ToolCall{
				ID:   fc.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      fc.Function,
					Arguments: fc.Arguments,
				},
			})
		}
		adapterReq.Messages[i] = adpMsg
	}

	for i, tool := range req.Tools {
		t := Tool{
			Type: tool.Type,
			Function: ToolFunc{
				Description: tool.Function.Description,
				Name:        tool.Function.Name,
				Parameters: ToolFuncParameters{
					Type:       tool.Function.Parameters.Type,
					Properties: make(map[string]ToolFuncParameterDetail),
				},
				Required: []string{},
			},
		}

		for paramName, paramDetail := range tool.Function.Parameters.Properties {
			t.Function.Parameters.Properties[paramName] = ToolFuncParameterDetail{
				Type:        paramDetail.Type,
				Description: paramDetail.Description,
			}
			if paramDetail.Required {
				t.Function.Required = append(t.Function.Required, paramName)
			}
		}
		adapterReq.Tools[i] = t
	}
	return adapterReq
}

// InitLLMClient initializes the LLMClient dependency
type InitLLMClient struct {
	HttpClient *http.Client `resolve:""`
	LLMHost    string       `config:"LLM_MODEL_HOST"`
}

// Initialize registers the LLMClient
func (i InitLLMClient) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.LLMClient](NewLLMClientAdapter(
		NewDRMAPIClient(i.LLMHost, "", i.HttpClient),
	))
	return ctx, nil
}
