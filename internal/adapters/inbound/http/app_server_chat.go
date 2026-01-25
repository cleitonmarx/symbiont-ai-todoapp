package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/openapi"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
)

func (api TodoAppServer) ClearChatMessages(w http.ResponseWriter, r *http.Request) {
	err := api.DeleteConversationUseCase.Execute(r.Context())
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (api TodoAppServer) ListChatMessages(w http.ResponseWriter, r *http.Request, params openapi.ListChatMessagesParams) {
	messages, hasMore, err := api.ListChatMessagesUseCase.Query(r.Context(), params.Page, params.Pagesize)
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	resp := openapi.ChatHistoryResp{
		ConversationId: domain.GlobalConversationID,
		Messages:       []openapi.ChatMessage{},
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
		resp.Messages = append(resp.Messages, openapi.ChatMessage{
			Id:        msg.ID,
			Role:      openapi.ChatMessageRole(msg.ChatRole),
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}

	respondJSON(w, http.StatusOK, resp)

}

func (api TodoAppServer) StreamChat(w http.ResponseWriter, r *http.Request) {
	req := openapi.StreamChatJSONRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, openapi.ErrorResp{
			Error: openapi.Error{
				Code:    openapi.BADREQUEST,
				Message: "invalid request body",
			},
		})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		errResp := openapi.ErrorResp{
			Error: openapi.Error{
				Code:    openapi.INTERNALERROR,
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

	err := api.StreamChatUseCase.Execute(r.Context(), req.Message, func(eventType domain.LLMStreamEventType, data any) error {
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
	})
	if err != nil {
		api.Logger.Printf("StreamChat: error during streaming: %v", err)
		respondError(w, toOpenAPIError(err))
	}
}
