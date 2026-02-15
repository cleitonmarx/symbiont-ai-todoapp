package http

import (
	"net/http"
)

// GetBoardSummary returns a summary of the current state of the board,
// including number of tasks, completed tasks, and pending tasks
// (GET /api/board/summary)
func (api TodoAppServer) GetBoardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := api.GetBoardSummaryUseCase.Query(r.Context())
	if err != nil {
		respondError(w, toError(err))
		return
	}

	respondJSON(w, http.StatusOK, toBoardSummary(summary))
}
