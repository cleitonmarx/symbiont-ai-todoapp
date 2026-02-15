package http

import (
	"encoding/json"
	"net/http"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// List conversations for the user
// (GET /api/conversations)
func (api TodoAppServer) ListConversations(w http.ResponseWriter, r *http.Request, params gen.ListConversationsParams) {
	conversations, hasMore, err := api.ListConversationsUseCase.Query(r.Context(), params.Page, params.PageSize)
	if err != nil {
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

// Delete a conversation
// (DELETE /api/conversations/{conversation_id})
func (api TodoAppServer) DeleteConversation(w http.ResponseWriter, r *http.Request, conversationId openapi_types.UUID) {
	err := api.DeleteConversationUseCase.Execute(r.Context(), conversationId)
	if err != nil {
		respondError(w, toError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Update conversation
// (PATCH /api/conversations/{conversation_id})
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

	updatedConversation, err := api.UpdateConversationUseCase.Execute(r.Context(), conversationId, req.Title)
	if err != nil {
		respondError(w, toError(err))
		return
	}

	resp := toConversation(updatedConversation)
	respondJSON(w, http.StatusOK, resp)
}
