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
		dataBytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "event: %s\n", eventType)
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
