package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LLMToolManager manages a collection of LLM tools.
type LLMToolManager struct {
	tools map[string]domain.LLMTool
}

// NewLLMToolManager creates a new LLMToolManager with the provided tools.
func NewLLMToolManager(tools ...domain.LLMTool) LLMToolManager {
	toolMap := make(map[string]domain.LLMTool)
	for _, tool := range tools {
		toolMap[tool.Definition().Function.Name] = tool
	}
	return LLMToolManager{
		tools: toolMap,
	}
}

// StatusMessage returns a status message about the tool execution.
func (m LLMToolManager) StatusMessage(functionName string) string {
	if tool, ok := m.tools[functionName]; ok {
		if msg := tool.StatusMessage(); msg != "" {
			return msg
		}
	}
	return "⏳ Processing request...\n\n"
}

// List returns all registered LLM tools.
func (ctr LLMToolManager) List() []domain.LLMToolDefinition {
	toolList := make([]domain.LLMToolDefinition, 0, len(ctr.tools))
	for _, tool := range ctr.tools {
		toolList = append(toolList, tool.Definition())
	}
	return toolList
}

// Call invokes the appropriate tool based on the function call.
func (ctr LLMToolManager) Call(ctx context.Context, call domain.LLMStreamEventToolCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	spanCtx, span := telemetry.Start(ctx,
		trace.WithAttributes(
			attribute.String("tool.function", call.Function),
		),
	)
	defer span.End()
	tool, exists := ctr.tools[call.Function]
	if !exists {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"unknown_tool","details":"Tool '%s' is not registered."}`, call.Function),
		}
	}
	return tool.Call(spanCtx, call, conversationHistory)
}

// NewTodoFetcherTool creates a new instance of TodoFetcherTool.
func NewTodoFetcherTool(repo domain.TodoRepository, llmCli domain.LLMClient, timeProvider domain.CurrentTimeProvider, llmEmbeddingModel string) TodoFetcherTool {
	return TodoFetcherTool{
		repo:              repo,
		timeProvider:      timeProvider,
		llmCli:            llmCli,
		llmEmbeddingModel: llmEmbeddingModel,
	}
}

// TodoFetcherTool is an LLM tool for fetching todos.
type TodoFetcherTool struct {
	repo              domain.TodoRepository
	timeProvider      domain.CurrentTimeProvider
	llmCli            domain.LLMClient
	llmEmbeddingModel string
}

// StatusMessage returns a status message about the tool execution.
func (t TodoFetcherTool) StatusMessage() string {
	return "🔎 Fetching todos...\n\n"
}

// Tool returns the LLMTool definition for the TodoFetcherTool.
func (lft TodoFetcherTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "fetch_todos",
			Description: "List existing todos with explicit pagination. Always pass page and page_size, start with page=1, and use returned next_page to keep fetching when full coverage is needed. Send a strict JSON object using only: page, page_size, status, search_term, sort_by, due_after, due_before. page and page_size must be positive integers. status must be OPEN or DONE. sort_by must be one of: dueDateAsc, dueDateDesc, createdAtAsc, createdAtDesc, similarityAsc, similarityDesc (use similarity sort only with search_term). due_after and due_before must be provided together in YYYY-MM-DD format. Avoid repeated identical calls. Valid: {\"page\":1,\"page_size\":10,\"status\":\"OPEN\"}. Invalid: {\"page\":\"1\",\"note\":\"x\"}.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
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
					"search_term": {
						Type:        "string",
						Description: "Optional search text used to find similar todos (e.g., dentist, groceries).",
						Required:    false,
					},
					"sort_by": {
						Type:        "string",
						Description: "Optional sort. Allowed: dueDateAsc, dueDateDesc, createdAtAsc, createdAtDesc, similarityAsc, similarityDesc. Use similarity sort only with search_term.",
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
		},
	}
}

// Call executes the TodoFetcherTool with the provided function call.
func (lft TodoFetcherTool) Call(ctx context.Context, call domain.LLMStreamEventToolCall, _ []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		Page       int     `json:"page"`
		PageSize   int     `json:"page_size"`
		Status     *string `json:"status"`
		SearchTerm *string `json:"search_term"`
		SortBy     *string `json:"sort_by"`
		DueAfter   *string `json:"due_after"`
		DueBefore  *string `json:"due_before"`
	}{
		Page:     1,  // default page
		PageSize: 10, // default page size
	}

	exampleArgs := `{"page":1,"page_size":50,"status":"OPEN", "search_term":"dentist appointments", "sort_by":"dueDateAsc", "due_after":"2026-04-01", "due_before":"2026-04-30"}`

	err := unmarshalToolArguments(call.Arguments, &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	opts := []domain.ListTodoOptions{}

	if params.SearchTerm != nil && *params.SearchTerm != "" {
		resp, err := lft.llmCli.Embed(ctx, lft.llmEmbeddingModel, *params.SearchTerm)
		if err != nil {
			return domain.LLMChatMessage{
				Role:    domain.ChatRole_Tool,
				Content: fmt.Sprintf(`{"error":"embedding_error","details":"%s"}`, err.Error()),
			}
		}
		RecordLLMTokensEmbedding(ctx, resp.TotalTokens)
		opts = append(opts, domain.WithEmbedding(resp.Embedding))
	}

	if params.Status != nil {
		opts = append(opts, domain.WithStatus(domain.TodoStatus(*params.Status)))
	}
	if params.SortBy != nil {
		opts = append(opts, domain.WithSortBy(*params.SortBy))
	}
	if params.DueAfter != nil && params.DueBefore != nil {
		now := lft.timeProvider.Now()
		dueAfter, ok := domain.ExtractTimeFromText(*params.DueAfter, now, now.Location())
		if !ok {
			return domain.LLMChatMessage{
				Role:    domain.ChatRole_Tool,
				Content: `{"error":"invalid_due_after","details":"Could not parse due_after date."}`,
			}
		}
		dueBefore, ok := domain.ExtractTimeFromText(*params.DueBefore, now, now.Location())
		if !ok {
			return domain.LLMChatMessage{
				Role:    domain.ChatRole_Tool,
				Content: `{"error":"invalid_due_before","details":"Could not parse due_before date."}`,
			}
		}

		opts = append(opts, domain.WithDueDateRange(dueAfter, dueBefore))
	}

	todos, hasMore, err := lft.repo.ListTodos(ctx, params.Page, params.PageSize, opts...)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"list_todos_error","details":"%s"}`, err.Error()),
		}
	}

	if len(todos) == 0 {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: `{"error":"no_todos_found","details":"No todos matched your search."}`,
		}
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
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"marshal_error","details":"%s"}`, err.Error()),
		}
	}

	return domain.LLMChatMessage{
		Role:    domain.ChatRole_Tool,
		Content: string(content),
	}
}

// TodoCreatorTool is an LLM tool for creating todos.
type TodoCreatorTool struct {
	uow          domain.UnitOfWork
	creator      TodoCreator
	timeProvider domain.CurrentTimeProvider
}

// NewTodoCreatorTool creates a new instance of TodoCreatorTool.
func NewTodoCreatorTool(uow domain.UnitOfWork, creator TodoCreator, timeProvider domain.CurrentTimeProvider) TodoCreatorTool {
	return TodoCreatorTool{
		uow:          uow,
		creator:      creator,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoCreatorTool) StatusMessage() string {
	return "📝 Creating your todo...\n\n"
}

// Tool returns the LLMTool definition for the TodoCreatorTool.
func (tct TodoCreatorTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "create_todo",
			Description: "Create exactly one todo. Required keys: title (string) and due_date (YYYY-MM-DD). No extra keys. Valid: {\"title\":\"Pay rent\",\"due_date\":\"2026-04-30\"}. Invalid: {\"title\":\"Pay rent\",\"due\":\"tomorrow\",\"priority\":\"high\"}.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"title": {
						Type:        "string",
						Description: "Todo title. REQUIRED.",
						Required:    true,
					},
					"due_date": {
						Type:        "string",
						Description: "Due date. REQUIRED. Use YYYY-MM-DD.",
						Required:    true,
					},
				},
			},
		},
	}
}

// Call executes the TodoCreatorTool with the provided function call.
func (tct TodoCreatorTool) Call(ctx context.Context, call domain.LLMStreamEventToolCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		Title   string `json:"title"`
		DueDate string `json:"due_date"`
	}{}

	exampleArgs := `{"title":"Pay rent","due_date":"2026-04-30"}`

	err := unmarshalToolArguments(call.Arguments, &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	now := tct.timeProvider.Now()
	dueDate, found := extractDateParam(params.DueDate, conversationHistory, now)
	if !found {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_due_date","details":"Due date cannot be empty. ISO 8601 string is required.", "example":%s}`, exampleArgs),
		}
	}

	var todo domain.Todo
	err = tct.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		td, err := tct.creator.Create(ctx, uow, params.Title, dueDate)
		if err != nil {
			return err
		}
		todo = td
		return nil
	})
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"create_todo_error","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	return domain.LLMChatMessage{
		Role:    domain.ChatRole_Tool,
		Content: "Your todo was created successfully! Created todo: " + todo.ToLLMInput(),
	}
}

// TodoMetaUpdaterTool is an LLM tool for updating todos.
type TodoMetaUpdaterTool struct {
	uow     domain.UnitOfWork
	updater TodoUpdater
}

// NewTodoMetaUpdaterTool creates a new instance of TodoMetaUpdaterTool.
func NewTodoMetaUpdaterTool(uow domain.UnitOfWork, updater TodoUpdater) TodoMetaUpdaterTool {
	return TodoMetaUpdaterTool{
		uow:     uow,
		updater: updater,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoMetaUpdaterTool) StatusMessage() string {
	return "✏️ Updating your todo...\n\n"
}

// Tool returns the LLMTool definition for the TodoMetaUpdaterTool.
func (tut TodoMetaUpdaterTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "update_todo",
			Description: "Update metadata for exactly one existing todo. Required key: id (UUID). Optional keys: title and status. Use this tool only for title/status changes (never due date). status must be OPEN or DONE. No extra keys. Valid: {\"id\":\"<uuid>\",\"status\":\"DONE\"}. Invalid: {\"id\":\"<uuid>\",\"due_date\":\"2026-04-30\"}.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"id": {
						Type:        "string",
						Description: "Todo UUID. REQUIRED.",
						Required:    true,
					},
					"title": {
						Type:        "string",
						Description: "New title (optional).",
						Required:    false,
					},
					"status": {
						Type:        "string",
						Description: "OPEN or DONE (optional).",
						Required:    false,
					},
				},
			},
		},
	}
}

// Call executes the TodoMetaUpdaterTool with the provided function call.
func (tut TodoMetaUpdaterTool) Call(ctx context.Context, call domain.LLMStreamEventToolCall, _ []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		ID     string  `json:"id"`
		Title  *string `json:"title"`
		Status *string `json:"status"`
	}{}

	exampleArgs := `{"id":"<uuid>","status":"DONE", "title":"New title"}`

	err := unmarshalToolArguments(call.Arguments, &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	todoID, err := uuid.Parse(params.ID)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_todo_id","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	var todo domain.Todo
	err = tut.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		td, err := tut.updater.Update(ctx, uow, todoID, params.Title, (*domain.TodoStatus)(params.Status), nil)
		if err != nil {
			return err
		}
		todo = td
		return nil
	})

	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"update_todo_error","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	return domain.LLMChatMessage{
		Role:    domain.ChatRole_Tool,
		Content: "Your todo was updated successfully! Updated todo: " + todo.ToLLMInput(),
	}
}

type TodoDueDateUpdaterTool struct {
	uow          domain.UnitOfWork
	updater      TodoUpdater
	timeProvider domain.CurrentTimeProvider
}

// NewTodoDueDateUpdaterTool creates a new instance of TodoDueDateUpdaterTool.
func NewTodoDueDateUpdaterTool(uow domain.UnitOfWork, updater TodoUpdater, timeProvider domain.CurrentTimeProvider) TodoDueDateUpdaterTool {
	return TodoDueDateUpdaterTool{
		uow:          uow,
		updater:      updater,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoDueDateUpdaterTool) StatusMessage() string {
	return "📅 Updating the due date...\n\n"
}

// Tool returns the LLMTool definition for the TodoDueDateUpdaterTool.
func (tdut TodoDueDateUpdaterTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "update_todo_due_date",
			Description: "Update due date for exactly one existing todo. Required keys: id (UUID string) and due_date (YYYY-MM-DD). Use this tool only for due-date changes. No extra keys. Valid: {\"id\":\"<uuid>\",\"due_date\":\"2026-04-30\"}. Invalid: {\"id\":\"<uuid>\",\"status\":\"DONE\"}.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"id": {
						Type:        "string",
						Description: "Todo UUID. REQUIRED.",
						Required:    true,
					},
					"due_date": {
						Type:        "string",
						Description: "Due date. REQUIRED. Use YYYY-MM-DD.",
						Required:    true,
					},
				},
			},
		},
	}
}

// Call executes the TodoDueDateUpdaterTool with the provided function call.
func (tdut TodoDueDateUpdaterTool) Call(ctx context.Context, call domain.LLMStreamEventToolCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		ID      uuid.UUID `json:"id"`
		DueDate string    `json:"due_date"`
	}{}

	exampleArgs := `{"id":"<uuid>","due_date":"2026-04-30"}`

	err := unmarshalToolArguments(call.Arguments, &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	if params.ID == uuid.Nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_todo_id","details":"Todo ID cannot be nil.", "example":%s}`, exampleArgs),
		}
	}

	now := tdut.timeProvider.Now()
	dueDate, found := extractDateParam(params.DueDate, conversationHistory, now)
	if !found {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_due_date","details":"Due date cannot be empty. ISO 8601 string is required.", "example":%s}`, exampleArgs),
		}
	}

	var todo domain.Todo
	err = tdut.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		td, err := tdut.updater.Update(ctx, uow, params.ID, nil, nil, &dueDate)
		if err != nil {
			return err
		}
		todo = td
		return nil
	})

	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"update_due_date_error","details":"%s"}`, err.Error()),
		}
	}
	return domain.LLMChatMessage{
		Role:    domain.ChatRole_Tool,
		Content: fmt.Sprintf(`{"message":"The due date was updated successfully! Updated todo: %s"}`, todo.ToLLMInput()),
	}
}

type TodoDeleterTool struct {
	uow     domain.UnitOfWork
	deleter TodoDeleter
}

// NewTodoDeleterTool creates a new instance of TodoDeleterTool.
func NewTodoDeleterTool(uow domain.UnitOfWork, deleter TodoDeleter) TodoDeleterTool {
	return TodoDeleterTool{
		uow:     uow,
		deleter: deleter,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoDeleterTool) StatusMessage() string {
	return "🗑️ Deleting the todo...\n\n"
}

// Tool returns the LLMTool definition for the TodoDeleterTool.
func (tdt TodoDeleterTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "delete_todo",
			Description: "Delete exactly one todo by id. Required key: id (UUID string). No extra keys. If id is unknown, call fetch_todos first. Valid: {\"id\":\"<uuid>\"}. Invalid: {\"id\":\"<uuid>\",\"confirm\":true}.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"id": {
						Type:        "string",
						Description: "Todo UUID. REQUIRED.",
						Required:    true,
					},
				},
			},
		},
	}
}

// Call executes the TodoDeleterTool with the provided function call.
func (tdt TodoDeleterTool) Call(ctx context.Context, call domain.LLMStreamEventToolCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		ID uuid.UUID `json:"id"`
	}{}

	exampleArgs := `{"id":"<uuid>"}`

	err := unmarshalToolArguments(call.Arguments, &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	err = tdt.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		return tdt.deleter.Delete(ctx, uow, params.ID)
	})
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"delete_todo_error","details":"%s"}`, err.Error()),
		}
	}

	return domain.LLMChatMessage{
		Role:    domain.ChatRole_Tool,
		Content: `{"message":"The todo was deleted successfully!"}`,
	}
}

// extractDateParam tries to extract a date from the provided parameter
// or from the user message history.
func extractDateParam(param string, history []domain.LLMChatMessage, referenceDate time.Time) (time.Time, bool) {
	// First, try to extract from the provided parameter
	if dueDate, ok := domain.ExtractTimeFromText(param, referenceDate, referenceDate.Location()); ok {
		return dueDate, true
	}

	// Next, scan the message history for date phrases
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Role != domain.ChatRole_User {
			continue
		}
		if dueDate, ok := domain.ExtractTimeFromText(msg.Content, referenceDate, referenceDate.Location()); ok {
			return dueDate, true
		}
	}
	return time.Time{}, false
}

// unmarshalToolArguments unmarshals the tool arguments from a JSON string into
// the target struct, ensuring that only a single JSON object is present and that there are no unknown fields.
func unmarshalToolArguments(arguments string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}

	// Reject trailing JSON values after the first object.
	var extra any
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("tool arguments must contain a single JSON object")
}

type InitLLMToolRegistry struct {
	Uow            domain.UnitOfWork          `resolve:""`
	TodoCreator    TodoCreator                `resolve:""`
	TodoUpdater    TodoUpdater                `resolve:""`
	TodoDeleter    TodoDeleter                `resolve:""`
	TodoRepo       domain.TodoRepository      `resolve:""`
	LLMClient      domain.LLMClient           `resolve:""`
	TimeProvider   domain.CurrentTimeProvider `resolve:""`
	EmbeddingModel string                     `config:"LLM_EMBEDDING_MODEL"`
}

func (i InitLLMToolRegistry) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.LLMToolRegistry](NewLLMToolManager(
		NewTodoFetcherTool(
			i.TodoRepo,
			i.LLMClient,
			i.TimeProvider,
			i.EmbeddingModel,
		),
		NewTodoCreatorTool(
			i.Uow,
			i.TodoCreator,
			i.TimeProvider,
		),
		NewTodoMetaUpdaterTool(
			i.Uow,
			i.TodoUpdater,
		),
		NewTodoDueDateUpdaterTool(
			i.Uow,
			i.TodoUpdater,
			i.TimeProvider,
		),
		NewTodoDeleterTool(
			i.Uow,
			i.TodoDeleter,
		),
	))
	return ctx, nil
}
