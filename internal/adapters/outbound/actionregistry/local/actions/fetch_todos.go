package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/metrics"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
	"github.com/toon-format/toon-go"
)

// NewFetchTodosAction creates a new instance of FetchTodosAction.
func NewFetchTodosAction(repo todo.Repository, semanticEncoder semantic.Encoder, embeddingModel string) FetchTodosAction {
	return FetchTodosAction{
		repo:            repo,
		semanticEncoder: semanticEncoder,
		embeddingModel:  embeddingModel,
	}
}

// FetchTodosAction is an assistant action for fetching todos.
type FetchTodosAction struct {
	repo            todo.Repository
	semanticEncoder semantic.Encoder
	embeddingModel  string
}

// StatusMessage returns a status message about the action execution.
func (t FetchTodosAction) StatusMessage() string {
	return "🔎 Fetching todos..."
}

// Renderer reports that fetch_todos does not expose a deterministic renderer yet.
func (t FetchTodosAction) Renderer() (assistant.ActionResultRenderer, bool) {
	return nil, false
}

// Definition returns the assistant action definition for FetchTodosAction.
func (lft FetchTodosAction) Definition() assistant.ActionDefinition {
	return assistant.ActionDefinition{
		Name:        "fetch_todos",
		Description: "Fetch todos with pagination and optional filters.",
		Input: assistant.ActionInput{
			Type: "object",
			Fields: map[string]assistant.ActionField{
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
					Description: "Optional status filter.",
					Required:    false,
					Enum:        []any{todo.Status_OPEN, todo.Status_DONE},
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
					Enum:        []any{"dueDateAsc", "dueDateDesc", "createdAtAsc", "createdAtDesc", "similarityAsc", "similarityDesc"},
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

// Execute executes FetchTodosAction.
func (lft FetchTodosAction) Execute(ctx context.Context, call assistant.ActionCall, _ []assistant.Message) assistant.Message {
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
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", fmt.Sprintf("failed to parse action input: %s", err.Error()), exampleArgs),
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

	buildResult, err := todouc.NewSearchBuilder().
		WithStatus((*todo.Status)(params.Status)).
		WithDueDateRange(dueAfterTime, dueBeforeTime).
		WithSortBy(params.SortBy).
		WithTitleContains(params.SearchByTitle).
		WithSimilaritySearch(params.SearchBySimilarity).
		Build(ctx, lft.semanticEncoder, lft.embeddingModel)
	if err != nil {
		code := mapTodoFilterBuildErrCode(err)
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError(code, err.Error(), exampleArgs),
		}
	}
	if buildResult.EmbeddingTotalTokens > 0 {
		metrics.RecordLLMTokensEmbedding(ctx, buildResult.EmbeddingTotalTokens)
	}

	todos, hasMore, err := lft.repo.ListTodos(ctx, params.Page, params.PageSize, buildResult.Options...)
	if err != nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("list_todos_error", fmt.Sprintf("failed to list todos:%s", err.Error()), exampleArgs),
		}
	}

	if len(todos) == 0 {
		todos = []todo.Todo{}
	}

	type result struct {
		ID      string `toon:"id"`
		Title   string `toon:"title"`
		DueDate string `toon:"due_date"`
		Status  string `toon:"status"`
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
	content, err := toon.Marshal(output)
	if err != nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("marshal_error", err.Error(), ""),
		}
	}

	return assistant.Message{
		Role:         assistant.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      string(content),
	}
}
