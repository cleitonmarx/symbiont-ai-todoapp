package http

import (
	"encoding/json"
	"net/http"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.opentelemetry.io/otel/trace"
)

// ListConversations lists conversations for the user.
// (GET /api/v1/conversations)
func (api TodoAppServer) ListConversations(w http.ResponseWriter, r *http.Request, params gen.ListConversationsParams) {
	ctx := r.Context()
	conversations, hasMore, err := api.ListConversationsUseCase.Query(ctx, params.Page, params.PageSize)
	if telemetry.IsErrorRecorded(trace.SpanFromContext(ctx), err) {
		api.Logger.Printf("Error listing conversations: %v", err)
		respondError(w, toError(err))
		return
	}

	resp := gen.ConversationListResp{
		Conversations: make([]gen.Conversation, len(conversations)),
		Page:          params.Page,
	}

	for i, c := range conversations {
		resp.Conversations[i] = toConversation(c)
	}
	if hasMore {
		nextPage := params.Page + 1
		resp.NextPage = &nextPage
	}
	if params.Page > 1 {
		prevPage := params.Page - 1
		resp.PreviousPage = &prevPage
	}

	respondJSON(w, http.StatusOK, resp)
}

// DeleteConversation deletes a conversation.
// (DELETE /api/v1/conversations/{conversation_id})
func (api TodoAppServer) DeleteConversation(w http.ResponseWriter, r *http.Request, conversationId openapi_types.UUID) {
	ctx := r.Context()
	err := api.DeleteConversationUseCase.Execute(ctx, conversationId)
	if telemetry.IsErrorRecorded(trace.SpanFromContext(ctx), err) {
		api.Logger.Printf("Error deleting conversation: %v", err)
		respondError(w, toError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateConversation updates a conversation.
// (PATCH /api/v1/conversations/{conversation_id})
func (api TodoAppServer) UpdateConversation(w http.ResponseWriter, r *http.Request, conversationId openapi_types.UUID) {
	var req gen.UpdateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, gen.ErrorResp{
			Error: gen.Error{
				Code:    gen.BADREQUEST,
				Message: "invalid request body",
			},
		})
		return
	}

	ctx := r.Context()
	updatedConversation, err := api.UpdateConversationUseCase.Execute(ctx, conversationId, req.Title)
	if telemetry.IsErrorRecorded(trace.SpanFromContext(ctx), err) {
		api.Logger.Printf("Error updating conversation: %v", err)
		respondError(w, toError(err))
		return
	}

	respondJSON(w, http.StatusOK, toConversation(updatedConversation))
}
