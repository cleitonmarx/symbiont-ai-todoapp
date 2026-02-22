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
		Hints: domain.AssistantActionHints{
			UseWhen:   "Read/query intents: show, list, find, filter, search, overdue, refetch.",
			AvoidWhen: "Do not use for create/update/delete operations.",
			ArgRules:  "Use only allowed keys. status is OPEN or DONE. Use search_by_similarity OR search_by_title, not both. due_after and due_before must come together.",
		},
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
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
