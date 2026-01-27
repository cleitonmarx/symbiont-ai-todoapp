package http

import (
	"encoding/json"
	"net/http"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/gen"
)

func respondJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, err gen.ErrorResp) {
	statusCode := http.StatusInternalServerError
	switch err.Error.Code {
	case gen.BADREQUEST:
		statusCode = http.StatusBadRequest
	case gen.NOTFOUND:
		statusCode = http.StatusNotFound
	}
	respondJSON(w, statusCode, err)
}
