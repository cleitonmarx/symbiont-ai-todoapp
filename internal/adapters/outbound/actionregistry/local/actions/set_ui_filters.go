package actions

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
)

// SetUIFiltersAction is an assistant action for synchronizing UI filter state.
type SetUIFiltersAction struct{}

// NewSetUIFiltersAction creates a new instance of SetUIFiltersAction.
func NewSetUIFiltersAction() SetUIFiltersAction {
	return SetUIFiltersAction{}
}

// StatusMessage returns a status message about the tool execution.
func (t SetUIFiltersAction) StatusMessage() string {
	return "🎛️ Applying filters..."
}

// Renderer reports that set_ui_filters does not expose a deterministic renderer.
func (t SetUIFiltersAction) Renderer() (assistant.ActionResultRenderer, bool) {
	return nil, false
}

// Definition returns the assistant action definition for SetUIFiltersAction.
func (t SetUIFiltersAction) Definition() assistant.ActionDefinition {
	return assistant.ActionDefinition{
		Name:        "set_ui_filters",
		Description: "Set UI filter state for read/query views.",
		Input: assistant.ActionInput{
			Type: "object",
			Fields: map[string]assistant.ActionField{
				"page": {
					Type:        "integer",
					Description: "page number starting from 1. Optional.",
					Required:    false,
				},
				"page_size": {
					Type:        "integer",
					Description: "Items per page. Optional. Allowed values: 25, 50, 100.",
					Required:    false,
					Enum:        []any{25, 50, 100},
				},
				"status": {
					Type:        "string",
					Description: "status filter. Optional.",
					Required:    false,
					Enum:        []any{todo.Status_OPEN, todo.Status_DONE},
				},
				"search_by_similarity": {
					Type:        "string",
					Description: "semantic search query. Optional.",
					Required:    false,
				},
				"search_by_title": {
					Type:        "string",
					Description: "title contains query. Optional.",
					Required:    false,
				},
				"sort_by": {
					Type:        "string",
					Description: "Optional sort. Allowed: dueDateAsc, dueDateDesc, createdAtAsc, createdAtDesc, similarityAsc, similarityDesc. Use similarity sort only with search_by_similarity. similarityAsc returns most similar first.",
					Required:    false,
					Enum:        []any{"dueDateAsc", "dueDateDesc", "createdAtAsc", "createdAtDesc", "similarityAsc", "similarityDesc"},
				},
				"due_after": {
					Type:        "string",
					Description: "lower due-date bound in YYYY-MM-DD. Must be provided with due_before. Optional.",
					Required:    false,
					Format:      "date",
				},
				"due_before": {
					Type:        "string",
					Description: "upper due-date bound in YYYY-MM-DD. Must be provided with due_after. Optional.",
					Required:    false,
					Format:      "date",
				},
			},
		},
	}
}

// Execute executes SetUIFiltersAction.
func (t SetUIFiltersAction) Execute(_ context.Context, call assistant.ActionCall, _ []assistant.Message) assistant.Message {
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

	exampleArgs := `{"search_by_similarity":"buy milk","sort_by":"similarityAsc","page":1,"page_size":25}`

	if err := unmarshalActionInput(call.Input, &params); err != nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", err.Error(), exampleArgs),
		}
	}

	var dueAfterTime *time.Time
	var dueBeforeTime *time.Time
	if params.DueAfter != nil || params.DueBefore != nil {
		var errMsg *assistant.Message
		dueAfterTime, dueBeforeTime, errMsg = parseDueDateParams(params.DueAfter, params.DueBefore, exampleArgs)
		if errMsg != nil {
			errMsg.ActionCallID = &call.ID
			return *errMsg
		}
	}

	err := todouc.NewTodoSearchBuilder().
		WithStatus((*todo.Status)(params.Status)).
		WithDueDateRange(dueAfterTime, dueBeforeTime).
		WithSortBy(params.SortBy).
		WithTitleContains(params.SearchByTitle).
		WithSimilaritySearch(params.SearchBySimilarity).
		Validate()
	if err != nil {
		code := mapTodoFilterBuildErrCode(err)
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError(code, err.Error(), exampleArgs),
		}
	}

	return assistant.Message{
		Role:         assistant.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      "ok",
	}
}
