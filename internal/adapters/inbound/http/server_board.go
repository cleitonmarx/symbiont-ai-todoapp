package http

import (
	"net/http"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"go.opentelemetry.io/otel/trace"
)

// GetBoardSummary returns a summary of the current state of the board,
// including number of tasks, completed tasks, and pending tasks
// (GET /api/v1/board/summary)
func (api TodoAppServer) GetBoardSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	summary, err := api.GetBoardSummaryUseCase.Query(ctx)
	if telemetry.IsErrorRecorded(trace.SpanFromContext(ctx), err) {
		api.Logger.Printf("Error getting board summary: %v", err)
		respondError(w, toError(err))
		return
	}

	respondJSON(w, http.StatusOK, toBoardSummary(summary))
}
