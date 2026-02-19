package actions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUIFiltersSetterAction(t *testing.T) {
	parseOutput := func(t *testing.T, content string) map[string]any {
		t.Helper()
		var output map[string]any
		err := json.Unmarshal([]byte(content), &output)
		require.NoError(t, err)
		return output
	}

	requireFilters := func(t *testing.T, output map[string]any) map[string]any {
		t.Helper()
		filters, ok := output["filters"].(map[string]any)
		require.True(t, ok)
		return filters
	}

	tests := map[string]struct {
		functionCall domain.AssistantActionCall
		validateResp func(t *testing.T, resp domain.AssistantMessage)
	}{
		"set-ui-filters-success-similarity": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"status":"OPEN","search_by_similarity":"buy milk","sort_by":"similarityAsc","page":1,"page_size":10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				output := parseOutput(t, resp.Content)
				assert.Equal(t, "ui_filters_set", output["message"])
				filters := requireFilters(t, output)
				assert.Equal(t, "OPEN", filters["status"])
				assert.Equal(t, "buy milk", filters["search_query"])
				assert.Equal(t, "SIMILARITY", filters["search_type"])
				assert.Equal(t, "similarityAsc", filters["sort_by"])
			},
		},
		"set-ui-filters-success-title-search-with-normalization": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"status":" open ","search_by_title":"  quarterly report  ","sort_by":" dueDateDesc ","due_after":" 2026-02-01 ","due_before":" 2026-02-28 ","page":2,"page_size":25}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				output := parseOutput(t, resp.Content)
				assert.Equal(t, "ui_filters_set", output["message"])
				filters := requireFilters(t, output)
				assert.Equal(t, "OPEN", filters["status"])
				assert.Equal(t, "quarterly report", filters["search_query"])
				assert.Equal(t, "TITLE", filters["search_type"])
				assert.Equal(t, "dueDateDesc", filters["sort_by"])
				assert.Equal(t, "2026-02-01", filters["due_after"])
				assert.Equal(t, "2026-02-28", filters["due_before"])
				assert.Equal(t, float64(2), filters["page"])
				assert.Equal(t, float64(25), filters["page_size"])
			},
		},
		"set-ui-filters-success-empty-similarity-is-ignored": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"search_by_similarity":"   ","page":1,"page_size":10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				output := parseOutput(t, resp.Content)
				filters := requireFilters(t, output)
				assert.NotContains(t, filters, "search_query")
				assert.NotContains(t, filters, "search_type")
				assert.Equal(t, float64(1), filters["page"])
				assert.Equal(t, float64(10), filters["page_size"])
			},
		},
		"set-ui-filters-success-one-search-empty-other-search-used": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"search_by_similarity":"   ","search_by_title":"buy milk","page":1,"page_size":10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				output := parseOutput(t, resp.Content)
				filters := requireFilters(t, output)
				assert.Equal(t, "buy milk", filters["search_query"])
				assert.Equal(t, "TITLE", filters["search_type"])
			},
		},
		"set-ui-filters-invalid-status": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"status":"OPEN,DONE","page":1,"page_size":10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "status must be OPEN or DONE")
			},
		},
		"set-ui-filters-invalid-both-search-modes": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"search_by_similarity":"buy","search_by_title":"buy","page":1,"page_size":10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "either search_by_similarity or search_by_title")
			},
		},
		"set-ui-filters-invalid-unknown-field": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"status":"OPEN","unknown_field":"x"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "unknown field")
			},
		},
		"set-ui-filters-invalid-multiple-json-values": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"status":"OPEN"}{"page":1}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "single JSON object")
			},
		},
		"set-ui-filters-invalid-partial-due-range": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"due_after":"2026-02-01"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "due_after and due_before must be provided together")
			},
		},
		"set-ui-filters-invalid-due-after-format": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"due_after":"2026/02/01","due_before":"2026-02-28"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "due_after must be YYYY-MM-DD")
			},
		},
		"set-ui-filters-invalid-due-before-format": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"due_after":"2026-02-01","due_before":"2026/02/28"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "due_before must be YYYY-MM-DD")
			},
		},
		"set-ui-filters-invalid-page-boundary": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"page":0}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "page must be >= 1")
			},
		},
		"set-ui-filters-invalid-page-size-boundary": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"page_size":0}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "page_size must be >= 1")
			},
		},
		"set-ui-filters-invalid-sort-by": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"sort_by":"dueDateASC"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
				assert.Contains(t, resp.Content, "sort_by is invalid")
			},
		},
		"set-ui-filters-invalid-json": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `not-json`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			action := NewUIFiltersSetterAction()
			assert.NotEmpty(t, action.StatusMessage())

			definition := action.Definition()
			assert.Equal(t, "set_ui_filters", definition.Name)
			assert.NotEmpty(t, definition.Description)
			assert.NotEmpty(t, definition.Input)

			resp := action.Execute(context.Background(), tt.functionCall, []domain.AssistantMessage{})
			tt.validateResp(t, resp)
		})
	}
}
