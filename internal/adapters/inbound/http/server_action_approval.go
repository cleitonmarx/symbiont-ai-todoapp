package http

import (
	"encoding/json"
	"net/http"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/chat"
	"go.opentelemetry.io/otel/trace"
)

// SubmitActionApproval handles one human approval decision for an assistant action call.
// (POST /api/v1/chat/approvals)
func (api TodoAppServer) SubmitActionApproval(w http.ResponseWriter, r *http.Request) {
	var req gen.SubmitActionApprovalJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, gen.ErrorResp{
			Error: gen.Error{
				Code:    gen.BADREQUEST,
				Message: "invalid request body",
			},
		})
		return
	}

	actionName := ""
	if req.ActionName != nil {
		actionName = *req.ActionName
	}

	ctx := r.Context()
	err := api.SubmitActionApprovalUseCase.Execute(ctx, chat.SubmitActionApprovalInput{
		ConversationID: req.ConversationId,
		TurnID:         req.TurnId,
		ActionCallID:   req.ActionCallId,
		ActionName:     actionName,
		Status:         assistant.ChatMessageApprovalStatus(req.Status),
		Reason:         req.Reason,
	})
	if telemetry.IsErrorRecorded(trace.SpanFromContext(ctx), err) {
		api.Logger.Printf("Error submitting action approval: %v", err)
		respondError(w, toError(err))
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
