package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
)

// NewTodoFetcherAction creates a new instance of TodoFetcherAction.
func NewTodoFetcherAction(repo domain.TodoRepository, semanticEncoder domain.SemanticEncoder, timeProvider domain.CurrentTimeProvider, embeddingModel string) TodoFetcherAction {
	return TodoFetcherAction{
		repo:            repo,
		timeProvider:    timeProvider,
		semanticEncoder: semanticEncoder,
		embeddingModel:  embeddingModel,
	}
}

// TodoFetcherAction is an assistant action for fetching todos.
type TodoFetcherAction struct {
	repo            domain.TodoRepository
	timeProvider    domain.CurrentTimeProvider
	semanticEncoder domain.SemanticEncoder
	embeddingModel  string
}

// StatusMessage returns a status message about the action execution.
func (t TodoFetcherAction) StatusMessage() string {
	return "🔎 Fetching todos..."
}

// Definition returns the assistant action definition for TodoFetcherAction.
func (lft TodoFetcherAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "fetch_todos",
		Description: "List existing todos with pagination. Required keys: page (integer >= 1), page_size (integer >= 1). Optional keys: status, search_by_similarity, search_by_title, sort_by, due_after, due_before. Use strict JSON only (double quotes; booleans unquoted). status accepts exactly OPEN or DONE. Never use combined values such as OPEN,DONE; to include all statuses, omit status. sort_by accepts: dueDateAsc, dueDateDesc, createdAtAsc, createdAtDesc, similarityAsc, similarityDesc (use similarity sort only with search_by_similarity). due_after and due_before must be provided together in YYYY-MM-DD format. Valid query template: {\"page\":1,\"page_size\":10,\"search_by_similarity\":\"buy milk\",\"sort_by\":\"similarityAsc\"}. Valid status template: {\"page\":1,\"page_size\":10,\"status\":\"OPEN\",\"sort_by\":\"dueDateAsc\"}. Invalid: {\"page\":1,\"page_size\":10,\"status\":\"OPEN,DONE\"}.",
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"page": {
					Type:        "integer",
					Description: "Page number starting from 1. REQUIRED on every fetch_todos call. Integer only.",
					Required:    true,
				},
				"page_size": {
					Type:        "integer",
					Description: "Items per page. REQUIRED on every fetch_todos call. Positive integer only.",
					Required:    true,
				},
				"status": {
					Type:        "string",
					Description: "Optional status filter. Allowed values: OPEN or DONE.",
					Required:    false,
				},
				"search_by_similarity": {
					Type:        "string",
					Description: "Optional semantic search text used to find similar todos (e.g., dentist, groceries). Generally should be used together with similarityAsc.",
					Required:    false,
				},
				"search_by_title": {
					Type:        "string",
					Description: "Optional text filter to find todos whose title contains the specified substring (case-insensitive).",
					Required:    false,
				},
				"sort_by": {
					Type:        "string",
					Description: "Optional sort. Allowed: dueDateAsc, dueDateDesc, createdAtAsc, createdAtDesc, similarityAsc, similarityDesc. Use similarity sort only with search_by_similarity. similarityAsc returns most similar first.",
					Required:    false,
				},
				"due_after": {
					Type:        "string",
					Description: "Optional lower due-date bound in YYYY-MM-DD. Must be provided together with due_before.",
					Required:    false,
				},
				"due_before": {
					Type:        "string",
					Description: "Optional upper due-date bound in YYYY-MM-DD. Must be provided together with due_after and should not be earlier than due_after.",
					Required:    false,
				},
			},
		},
	}
}

// Execute executes TodoFetcherAction.
func (lft TodoFetcherAction) Execute(ctx context.Context, call domain.AssistantActionCall, _ []domain.AssistantMessage) domain.AssistantMessage {
	params := struct {
		Page               int     `json:"page"`
		PageSize           int     `json:"page_size"`
		Status             *string `json:"status"`
		SearchBySimilarity *string `json:"search_by_similarity"`
		SearchByTitle      *string `json:"search_by_title"`
		SortBy             *string `json:"sort_by"`
		DueAfter           *string `json:"due_after"`
		DueBefore          *string `json:"due_before"`
	}{
		Page:     1,  // default page
		PageSize: 10, // default page size
	}

	exampleArgs := `{"page":1,"page_size":10,"search_by_similarity":"dinner","sort_by":"similarityAsc"}`

	err := unmarshalActionInput(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	var dueAfterTime *time.Time
	var dueBeforeTime *time.Time
	if params.DueAfter != nil || params.DueBefore != nil {
		now := lft.timeProvider.Now()
		if params.DueAfter != nil {
			dueAfter, ok := domain.ExtractTimeFromText(*params.DueAfter, now, now.Location())
			if !ok {
				return domain.AssistantMessage{
					Role:         domain.ChatRole_Tool,
					ActionCallID: &call.ID,
					Content:      `{"error":"invalid_due_after","details":"Could not parse due_after date."}`,
				}
			}
			dueAfterTime = &dueAfter
		}
		if params.DueBefore != nil {
			dueBefore, ok := domain.ExtractTimeFromText(*params.DueBefore, now, now.Location())
			if !ok {
				return domain.AssistantMessage{
					Role:         domain.ChatRole_Tool,
					ActionCallID: &call.ID,
					Content:      `{"error":"invalid_due_before","details":"Could not parse due_before date."}`,
				}
			}
			dueBeforeTime = &dueBefore
		}
	}

	buildResult, err := usecases.NewTodoSearchBuilder(lft.semanticEncoder, lft.embeddingModel).
		WithStatus((*domain.TodoStatus)(params.Status)).
		WithDueDateRange(dueAfterTime, dueBeforeTime).
		WithSortBy(params.SortBy).
		WithTitleContains(params.SearchByTitle).
		WithSimilaritySearch(params.SearchBySimilarity).
		Build(ctx)
	if err != nil {
		code := mapTodoFilterBuildErrCode(err)
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"%s","details":"%s"}`, code, err.Error()),
		}
	}
	if buildResult.EmbeddingTotalTokens > 0 {
		usecases.RecordLLMTokensEmbedding(ctx, buildResult.EmbeddingTotalTokens)
	}

	todos, hasMore, err := lft.repo.ListTodos(ctx, params.Page, params.PageSize, buildResult.Options...)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"list_todos_error","details":"%s"}`, err.Error()),
		}
	}

	if len(todos) == 0 {
		todos = []domain.Todo{}
	}

	type result struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		DueDate string `json:"due_date"`
		Status  string `json:"status"`
	}

	todosResult := make([]result, len(todos))
	for i, t := range todos {
		todosResult[i] = result{
			ID:      t.ID.String(),
			Title:   t.Title,
			DueDate: t.DueDate.Format(time.DateOnly),
			Status:  string(t.Status),
		}
	}

	var nextPage *int
	if hasMore {
		nxt := params.Page + 1
		nextPage = &nxt
	}

	output := map[string]any{
		"todos":     todosResult,
		"next_page": nextPage,
	}
	content, err := json.Marshal(output)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"marshal_error","details":"%s"}`, err.Error()),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      string(content),
	}
}

// mapTodoFilterBuildErrCode maps errors from building todo search options to specific error codes for better client handling.
func mapTodoFilterBuildErrCode(err error) string {
	var validationErr *domain.ValidationErr
	if errors.As(err, &validationErr) {
		switch err.Error() {
		case "due_after and due_before must be provided together":
			return "invalid_due_range"
		case "due_after must be less than or equal to due_before":
			return "invalid_due_range"
		case "search_by_similarity is required when using similarity sorting":
			return "missing_search_by_similarity_for_similarity_sort"
		default:
			return "invalid_filters"
		}
	}
	return "embedding_error"
}
