package http

import (
	"net/http"
)

func (api TodoAppServer) GetBoardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := api.GetBoardSummaryUseCase.Query(r.Context())
	if err != nil {
		respondError(w, toError(err))
		return
	}

	respondJSON(w, http.StatusOK, toBoardSummary(summary))
}
