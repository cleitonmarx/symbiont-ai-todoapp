package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// UIFiltersSetterTool is an LLM tool for synchronizing UI filter state.
type UIFiltersSetterTool struct{}

// NewUIFiltersSetterTool creates a new instance of UIFiltersSetterTool.
func NewUIFiltersSetterTool() UIFiltersSetterTool {
	return UIFiltersSetterTool{}
}

// StatusMessage returns a status message about the tool execution.
func (t UIFiltersSetterTool) StatusMessage() string {
	return "🎛️ Applying filters..."
}

// Tool returns the LLMTool definition for the UIFiltersSetterTool.
func (t UIFiltersSetterTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "set_ui_filters",
			Description: "Set filters in the main UI. Use for read/query intents (show/list/find/filter/search/overdue/refetch). Strict JSON object only. Allowed keys: status, search_by_similarity, search_by_title, sort_by, due_after, due_before, page, page_size. status must be OPEN or DONE when provided. If user asks for all statuses, omit status. Use either search_by_similarity or search_by_title, not both. due_after and due_before must be provided together as YYYY-MM-DD.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"status": {
						Type:        "string",
						Description: "Optional status filter. Allowed: OPEN or DONE.",
						Required:    false,
					},
					"search_by_similarity": {
						Type:        "string",
						Description: "Optional semantic search query.",
						Required:    false,
					},
					"search_by_title": {
						Type:        "string",
						Description: "Optional title contains query.",
						Required:    false,
					},
					"sort_by": {
						Type:        "string",
						Description: "Optional sort. Allowed: dueDateAsc, dueDateDesc, createdAtAsc, createdAtDesc, similarityAsc, similarityDesc.",
						Required:    false,
					},
					"due_after": {
						Type:        "string",
						Description: "Optional lower due-date bound in YYYY-MM-DD. Must be provided with due_before.",
						Required:    false,
					},
					"due_before": {
						Type:        "string",
						Description: "Optional upper due-date bound in YYYY-MM-DD. Must be provided with due_after.",
						Required:    false,
					},
					"page": {
						Type:        "integer",
						Description: "Optional page number starting from 1. Default 1.",
						Required:    false,
					},
					"page_size": {
						Type:        "integer",
						Description: "Optional page size. Default 10. Use 25, 50, or 100 for larger sizes.",
						Required:    false,
					},
				},
			},
		},
	}
}

// Call executes the UIFiltersSetterTool with the provided function call.
func (t UIFiltersSetterTool) Call(_ context.Context, call domain.LLMStreamEventToolCall, _ []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		Status             *string `json:"status"`
		SearchBySimilarity *string `json:"search_by_similarity"`
		SearchByTitle      *string `json:"search_by_title"`
		SortBy             *string `json:"sort_by"`
		DueAfter           *string `json:"due_after"`
		DueBefore          *string `json:"due_before"`
		Page               *int    `json:"page"`
		PageSize           *int    `json:"page_size"`
	}{}

	exampleArgs := `{"search_by_similarity":"buy milk","sort_by":"similarityAsc","page":1,"page_size":10}`

	if err := unmarshalToolArguments(call.Arguments, &params); err != nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"%s","example":%s}`, err.Error(), exampleArgs),
		}
	}

	if params.Status != nil {
		normalizedStatus := strings.ToUpper(strings.TrimSpace(*params.Status))
		if normalizedStatus != "OPEN" && normalizedStatus != "DONE" {
			return domain.LLMChatMessage{
				Role:       domain.ChatRole_Tool,
				ToolCallID: &call.ID,
				Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"status must be OPEN or DONE. Omit status for all.","example":%s}`, exampleArgs),
			}
		}
		params.Status = &normalizedStatus
	}

	if params.SearchBySimilarity != nil && strings.TrimSpace(*params.SearchBySimilarity) == "" {
		params.SearchBySimilarity = nil
	}
	if params.SearchByTitle != nil && strings.TrimSpace(*params.SearchByTitle) == "" {
		params.SearchByTitle = nil
	}
	if params.SearchBySimilarity != nil && params.SearchByTitle != nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"use either search_by_similarity or search_by_title, not both.","example":%s}`, exampleArgs),
		}
	}

	if params.SortBy != nil {
		sortBy := strings.TrimSpace(*params.SortBy)
		switch sortBy {
		case "dueDateAsc", "dueDateDesc", "createdAtAsc", "createdAtDesc", "similarityAsc", "similarityDesc":
			params.SortBy = &sortBy
		default:
			return domain.LLMChatMessage{
				Role:       domain.ChatRole_Tool,
				ToolCallID: &call.ID,
				Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"sort_by is invalid.","example":%s}`, exampleArgs),
			}
		}
	}

	if (params.DueAfter == nil) != (params.DueBefore == nil) {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"due_after and due_before must be provided together.","example":%s}`, exampleArgs),
		}
	}
	if params.DueAfter != nil && params.DueBefore != nil {
		dueAfter := strings.TrimSpace(*params.DueAfter)
		dueBefore := strings.TrimSpace(*params.DueBefore)
		if _, err := time.Parse(time.DateOnly, dueAfter); err != nil {
			return domain.LLMChatMessage{
				Role:       domain.ChatRole_Tool,
				ToolCallID: &call.ID,
				Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"due_after must be YYYY-MM-DD.","example":%s}`, exampleArgs),
			}
		}
		if _, err := time.Parse(time.DateOnly, dueBefore); err != nil {
			return domain.LLMChatMessage{
				Role:       domain.ChatRole_Tool,
				ToolCallID: &call.ID,
				Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"due_before must be YYYY-MM-DD.","example":%s}`, exampleArgs),
			}
		}
		params.DueAfter = &dueAfter
		params.DueBefore = &dueBefore
	}

	if params.Page != nil && *params.Page <= 0 {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"page must be >= 1.","example":%s}`, exampleArgs),
		}
	}
	if params.PageSize != nil && *params.PageSize <= 0 {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"page_size must be >= 1.","example":%s}`, exampleArgs),
		}
	}

	filters := map[string]any{}
	if params.Status != nil {
		filters["status"] = *params.Status
	}
	if params.SearchBySimilarity != nil {
		filters["search_query"] = strings.TrimSpace(*params.SearchBySimilarity)
		filters["search_type"] = "SIMILARITY"
	}
	if params.SearchByTitle != nil {
		filters["search_query"] = strings.TrimSpace(*params.SearchByTitle)
		filters["search_type"] = "TITLE"
	}
	if params.SortBy != nil {
		filters["sort_by"] = *params.SortBy
	}
	if params.DueAfter != nil {
		filters["due_after"] = *params.DueAfter
	}
	if params.DueBefore != nil {
		filters["due_before"] = *params.DueBefore
	}
	if params.Page != nil {
		filters["page"] = *params.Page
	}
	if params.PageSize != nil {
		filters["page_size"] = *params.PageSize
	}

	content, err := json.Marshal(map[string]any{
		"message": "ui_filters_set",
		"filters": filters,
	})
	if err != nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"marshal_error","details":"%s"}`, err.Error()),
		}
	}

	return domain.LLMChatMessage{
		Role:       domain.ChatRole_Tool,
		ToolCallID: &call.ID,
		Content:    string(content),
	}
}
