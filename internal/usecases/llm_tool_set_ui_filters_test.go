package usecases

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUIFiltersSetterTool(t *testing.T) {
	tests := map[string]struct {
		functionCall domain.LLMStreamEventToolCall
		validateResp func(t *testing.T, resp domain.LLMChatMessage)
	}{
		"set-ui-filters-success-similarity": {
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "set_ui_filters",
				Arguments: `{"status":"OPEN","search_by_similarity":"buy milk","sort_by":"similarityAsc","page":1,"page_size":10}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.Equal(t, "ui_filters_set", output["message"])
				filters, ok := output["filters"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "OPEN", filters["status"])
				assert.Equal(t, "buy milk", filters["search_query"])
				assert.Equal(t, "SIMILARITY", filters["search_type"])
				assert.Equal(t, "similarityAsc", filters["sort_by"])
			},
		},
		"set-ui-filters-invalid-status": {
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "set_ui_filters",
				Arguments: `{"status":"OPEN,DONE","page":1,"page_size":10}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "status must be OPEN or DONE")
			},
		},
		"set-ui-filters-invalid-both-search-modes": {
			functionCall: domain.LLMStreamEventToolCall{
				Function:  "set_ui_filters",
				Arguments: `{"search_by_similarity":"buy","search_by_title":"buy","page":1,"page_size":10}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "either search_by_similarity or search_by_title")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tool := NewUIFiltersSetterTool()
			resp := tool.Call(context.Background(), tt.functionCall, []domain.LLMChatMessage{})
			tt.validateResp(t, resp)
		})
	}
}
