package http

import (
	"net/http"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/openapi"
)

func (api TodoAppServer) GetBoardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := api.GetBoardSummaryUseCase.Query(r.Context())
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	resp := openapi.BoardSummary{
		Counts: openapi.TodoStatusCounts{
			DONE: summary.Content.Counts.Done,
			OPEN: summary.Content.Counts.Open,
		},
		NearDeadline: summary.Content.NearDeadline,
		NextUp:       []openapi.NextUpTodoItem{},
		Overdue:      summary.Content.Overdue,
		Summary:      summary.Content.Summary,
	}
	for _, item := range summary.Content.NextUp {
		resp.NextUp = append(resp.NextUp, openapi.NextUpTodoItem{
			Title:  item.Title,
			Reason: item.Reason,
		})
	}

	respondJSON(w, http.StatusOK, resp)
}
