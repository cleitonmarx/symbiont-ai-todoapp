package actions

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
)

// UIFiltersSetterAction is an assistant action for synchronizing UI filter state.
type UIFiltersSetterAction struct{}

// NewUIFiltersSetterAction creates a new instance of UIFiltersSetterAction.
func NewUIFiltersSetterAction() UIFiltersSetterAction {
	return UIFiltersSetterAction{}
}

// StatusMessage returns a status message about the tool execution.
func (t UIFiltersSetterAction) StatusMessage() string {
	return "🎛️ Applying filters..."
}

// Definition returns the assistant action definition for UIFiltersSetterAction.
func (t UIFiltersSetterAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "set_ui_filters",
		Description: "Set UI filter state for read/query views.",
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"page": {
					Type:        "integer",
					Description: "page number starting from 1. Optional.",
					Required:    false,
				},
				"page_size": {
					Type:        "integer",
					Description: "Items per page. Optional.",
					Required:    false,
					Enum:        []any{25, 50, 100},
				},
				"status": {
					Type:        "string",
					Description: "status filter. Optional.",
					Required:    false,
					Enum:        []any{domain.TodoStatus_OPEN, domain.TodoStatus_DONE},
				},
				"search_by_similarity": {
					Type:        "string",
					Description: "semantic search query. Optional`.",
					Required:    false,
				},
				"search_by_title": {
					Type:        "string",
					Description: "title contains query. Optional.",
					Required:    false,
				},
				"sort_by": {
					Type:        "string",
					Description: "sort option. Optional.",
					Required:    false,
					Enum:        []any{"dueDateAsc", "dueDateDesc", "createdAtAsc", "createdAtDesc", "similarityAsc", "similarityDesc"},
				},
				"due_after": {
					Type:        "string",
					Description: "lower due-date bound in YYYY-MM-DD. Must be provided with due_before. Optional.",
					Required:    false,
				},
				"due_before": {
					Type:        "string",
					Description: "upper due-date bound in YYYY-MM-DD. Must be provided with due_after. Optional.",
					Required:    false,
				},
			},
		},
	}
}

// Execute executes UIFiltersSetterAction.
func (t UIFiltersSetterAction) Execute(_ context.Context, call domain.AssistantActionCall, _ []domain.AssistantMessage) domain.AssistantMessage {
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

	if err := unmarshalActionInput(call.Input, &params); err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", err.Error(), exampleArgs),
		}
	}

	var dueAfterTime *time.Time
	var dueBeforeTime *time.Time
	if params.DueAfter != nil || params.DueBefore != nil {
		var errMsg *domain.AssistantMessage
		dueAfterTime, dueBeforeTime, errMsg = parseDueDateParams(params.DueAfter, params.DueBefore, exampleArgs)
		if errMsg != nil {
			errMsg.ActionCallID = &call.ID
			return *errMsg
		}
	}

	err := usecases.NewTodoSearchBuilder().
		WithStatus((*domain.TodoStatus)(params.Status)).
		WithDueDateRange(dueAfterTime, dueBeforeTime).
		WithSortBy(params.SortBy).
		WithTitleContains(params.SearchByTitle).
		WithSimilaritySearch(params.SearchBySimilarity).
		Validate()
	if err != nil {
		code := mapTodoFilterBuildErrCode(err)
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError(code, err.Error(), exampleArgs),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      "ok",
	}
}
