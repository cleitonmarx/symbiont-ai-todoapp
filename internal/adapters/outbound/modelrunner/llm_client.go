package modelrunner

import (
	"context"
	"errors"
	"net/http"
	"strings"
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

	adapterReq := ChatRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Messages:    make([]ChatMessage, len(req.Messages)),
	}

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

	adapterReq := ChatRequest{
		Model:    req.Model,
		Messages: make([]ChatMessage, len(req.Messages)),
	}

	for i, msg := range req.Messages {
		adapterReq.Messages[i] = ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

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

	var finalUsage *Usage
	estimatedPromptTokens := estimateTokenCount(adapterReq.Messages)

	// Stream chunks
	err := a.client.ChatStream(spanCtx, adapterReq, func(chunk StreamChunk) error {
		// Capture usage from chunk
		if chunk.Usage != nil {
			finalUsage = chunk.Usage
		}

		// Extract usage from timings if not provided
		if chunk.Timings != nil && finalUsage == nil {
			finalUsage = &Usage{
				PromptTokens:     chunk.Timings.PromptN,
				CompletionTokens: chunk.Timings.PredictedN,
				TotalTokens:      chunk.Timings.PromptN + chunk.Timings.PredictedN,
			}
		}

		// Send delta events
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if err := onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{
					Text: choice.Delta.Content,
				}); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Fallback for usage estimation
	if finalUsage == nil {
		finalUsage = &Usage{
			PromptTokens:     estimatedPromptTokens,
			CompletionTokens: 0,
			TotalTokens:      estimatedPromptTokens,
		}
	} else if finalUsage.PromptTokens < estimatedPromptTokens {
		finalUsage.PromptTokens = estimatedPromptTokens
		finalUsage.TotalTokens = finalUsage.PromptTokens + finalUsage.CompletionTokens
	}

	// Send done event
	done := domain.LLMStreamEventDone{
		AssistantMessageID: meta.AssistantMessageID.String(),
		CompletedAt:        time.Now().UTC().Format(time.RFC3339),
		Usage: &domain.LLMUsage{
			PromptTokens:     finalUsage.PromptTokens,
			CompletionTokens: finalUsage.CompletionTokens,
			TotalTokens:      finalUsage.TotalTokens,
		},
	}
	return onEvent(domain.LLMStreamEventType_Done, done)
}

// estimateTokenCount estimates tokens from messages
func estimateTokenCount(messages []ChatMessage) int {
	totalWords := 0
	for _, msg := range messages {
		totalWords += 4 // message overhead
		totalWords += len(strings.Fields(msg.Content))
	}
	return int(float64(totalWords) * 1.3)
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
