package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"go.opentelemetry.io/otel/trace"
)

// List chat messages for a conversation with pagination
// (GET /api/conversations/{conversation_id}/messages)
func (api TodoAppServer) ListChatMessages(w http.ResponseWriter, r *http.Request, params gen.ListChatMessagesParams) {
	messages, hasMore, err := api.ListChatMessagesUseCase.Query(r.Context(), params.ConversationId, params.Page, params.PageSize)
	if err != nil {
		respondError(w, toError(err))
		return
	}

	resp := gen.ChatHistoryResp{
		ConversationId: params.ConversationId,
		Messages:       []gen.ChatMessage{},
		Page:           params.Page,
	}
	if hasMore {
		nextPage := params.Page + 1
		resp.NextPage = &nextPage
	}
	if params.Page > 1 {
		prevPage := params.Page - 1
		resp.PreviousPage = &prevPage
	}

	for _, msg := range messages {
		resp.Messages = append(resp.Messages, toChatMessage(msg))
	}

	respondJSON(w, http.StatusOK, resp)

}

// StreamChat handles streaming assistant chat responses
// (POST /api/chat/stream)
func (api TodoAppServer) StreamChat(w http.ResponseWriter, r *http.Request) {
	req := gen.StreamChatJSONRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, gen.ErrorResp{
			Error: gen.Error{
				Code:    gen.BADREQUEST,
				Message: "invalid request body",
			},
		})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		errResp := gen.ErrorResp{
			Error: gen.Error{
				Code:    gen.INTERNALERROR,
				Message: "streaming not supported",
			},
		}
		respondError(w, errResp)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var options []usecases.StreamChatOption
	if req.ConversationId != nil {
		options = append(options, usecases.WithConversationID(*req.ConversationId))
	}

	err := api.StreamChatUseCase.Execute(r.Context(), req.Message, req.Model, func(eventType domain.AssistantEventType, data any) error {
		sseEventType, sseData, shouldHandle, mapErr := mapAssistantEventToSSE(eventType, data)
		if mapErr != nil {
			return mapErr
		}
		if !shouldHandle {
			return nil
		}

		dataBytes, err := json.Marshal(sseData)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, "event: %s\n", sseEventType)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, "data: %s\n\n", string(dataBytes))
		if err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}, options...)
	if telemetry.RecordErrorAndStatus(trace.SpanFromContext(r.Context()), err) &&
		!errors.Is(err, context.Canceled) {
		api.Logger.Printf("StreamChat: error during streaming: %v", err)
		respondError(w, toError(err))
	}
}

func mapAssistantEventToSSE(eventType domain.AssistantEventType, data any) (string, any, bool, error) {
	switch eventType {
	case domain.AssistantEventType_TurnStarted:
		started, ok := data.(domain.AssistantTurnStarted)
		if !ok {
			return "", nil, false, fmt.Errorf("unexpected assistant turn_started payload type: %T", data)
		}
		return string(domain.AssistantEventType_TurnStarted), sseTurnStartedEvent{
			ConversationID:      started.ConversationID,
			UserMessageID:       started.UserMessageID,
			AssistantMessageID:  started.AssistantMessageID,
			ConversationCreated: started.ConversationCreated,
		}, true, nil
	case domain.AssistantEventType_MessageDelta:
		delta, ok := data.(domain.AssistantMessageDelta)
		if !ok {
			return "", nil, false, fmt.Errorf("unexpected assistant message_delta payload type: %T", data)
		}
		return string(domain.AssistantEventType_MessageDelta), sseMessageDeltaEvent{
			Text: delta.Text,
		}, true, nil
	case domain.AssistantEventType_ActionStarted:
		call, ok := data.(domain.AssistantActionCall)
		if !ok {
			return "", nil, false, fmt.Errorf("unexpected assistant action_started payload type: %T", data)
		}
		return string(domain.AssistantEventType_ActionStarted), sseActionCallEvent{
			ID:    call.ID,
			Name:  call.Name,
			Input: call.Input,
			Text:  call.Text,
		}, true, nil
	case domain.AssistantEventType_ActionCompleted:
		completed, ok := data.(domain.AssistantActionCompleted)
		if !ok {
			return "", nil, false, fmt.Errorf("unexpected assistant action_completed payload type: %T", data)
		}
		return string(domain.AssistantEventType_ActionCompleted), sseActionCompletedEvent{
			ID:            completed.ID,
			Name:          completed.Name,
			Success:       completed.Success,
			Error:         completed.Error,
			ShouldRefetch: completed.ShouldRefetch,
		}, true, nil
	case domain.AssistantEventType_ActionRequested:
		call, ok := data.(domain.AssistantActionCall)
		if !ok {
			return "", nil, false, fmt.Errorf("unexpected assistant action_requested payload type: %T", data)
		}
		return string(domain.AssistantEventType_ActionRequested), sseActionCallEvent{
			ID:    call.ID,
			Name:  call.Name,
			Input: call.Input,
			Text:  call.Text,
		}, true, nil
	case domain.AssistantEventType_TurnCompleted:
		done, ok := data.(domain.AssistantTurnCompleted)
		if !ok {
			return "", nil, false, fmt.Errorf("unexpected assistant turn_completed payload type: %T", data)
		}
		return string(domain.AssistantEventType_TurnCompleted), sseTurnCompletedEvent{
			Usage:              done.Usage,
			AssistantMessageID: done.AssistantMessageID,
			CompletedAt:        done.CompletedAt,
		}, true, nil
	default:
		return "", nil, false, nil
	}
}

type sseTurnStartedEvent struct {
	ConversationID      any  `json:"conversation_id"`
	UserMessageID       any  `json:"user_message_id"`
	AssistantMessageID  any  `json:"assistant_message_id"`
	ConversationCreated bool `json:"conversation_created"`
}

type sseMessageDeltaEvent struct {
	Text string `json:"text"`
}

type sseActionCallEvent struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
	Text  string `json:"text"`
}

type sseActionCompletedEvent struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Success       bool    `json:"success"`
	Error         *string `json:"error,omitempty"`
	ShouldRefetch bool    `json:"should_refetch"`
}

type sseTurnCompletedEvent struct {
	Usage              domain.AssistantUsage `json:"usage"`
	AssistantMessageID string                `json:"assistant_message_id"`
	CompletedAt        string                `json:"completed_at"`
}

// ListAvailableModels returns the list of available assistant models for chat
// (GET /api/models)
func (api TodoAppServer) ListAvailableModels(w http.ResponseWriter, r *http.Request) {
	models, err := api.ListAvailableModelsUseCase.Query(r.Context())
	if err != nil {
		respondError(w, toError(err))
		return
	}

	rp := gen.ModelListResp{}
	for _, m := range models {
		if m.Kind != domain.ModelKindAssistant {
			continue
		}
		rp.Models = append(rp.Models, m.Name)
	}

	respondJSON(w, http.StatusOK, rp)
}
