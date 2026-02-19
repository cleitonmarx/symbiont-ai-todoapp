package actions

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestUIFiltersSetterAction(t *testing.T) {
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
				assert.Equal(t, "ok", resp.Content)
			},
		},
		"set-ui-filters-success-one-search-empty-other-search-used": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"search_by_similarity":"   ","search_by_title":"buy milk","page":1,"page_size":10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Equal(t, "ok", resp.Content)
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
				assert.Contains(t, resp.Content, "invalid_due_range")
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
				assert.Contains(t, resp.Content, "invalid_due_after")
				assert.Contains(t, resp.Content, "could not parse due_after date")
			},
		},
		"set-ui-filters-invalid-due-before-format": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"due_after":"2026-02-01","due_before":"2026/02/28"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_before")
				assert.Contains(t, resp.Content, "could not parse due_before date")
			},
		},
		"set-ui-filters-invalid-sort-by": {
			functionCall: domain.AssistantActionCall{
				Name:  "set_ui_filters",
				Input: `{"sort_by":"dueDateASC"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_sort_by")
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
