package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
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
func (ctr LLMToolManager) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	spanCtx, span := tracing.Start(ctx)
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
func NewTodoFetcherTool(repo domain.TodoRepository, llmCli domain.LLMClient, llmEmbeddingModel string) TodoFetcherTool {
	return TodoFetcherTool{
		repo:              repo,
		llmCli:            llmCli,
		llmEmbeddingModel: llmEmbeddingModel,
	}
}

// TodoFetcherTool is an LLM tool for fetching todos.
type TodoFetcherTool struct {
	repo              domain.TodoRepository
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
			Description: "Fetch todos by keyword. Use ONLY when you need to find existing items. Provide all parameters exactly as specified (integers for page/page_size, string for search_term). Do NOT include extra keys. Do NOT call repeatedly with the same parameters.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"page": {
						Type:        "integer",
						Description: "Page number (starting from 1). REQUIRED. Integer only.",
						Required:    true,
					},
					"page_size": {
						Type:        "integer",
						Description: "Items per page (1–30). REQUIRED. Integer only.",
						Required:    true,
					},
					"status": {
						Type:        "string",
						Description: "Filter by status: OPEN or DONE (optional).",
						Required:    false,
					},
					"search_term": {
						Type:        "string",
						Description: "Keyword/phrase to search (e.g., 'dentist', 'shopping', 'groceries').",
						Required:    false,
					},
					"sort_by": {
						Type:        "string",
						Description: "Sort by 'dueDateAsc', 'dueDateDesc', 'createdAtAsc', 'createdAtDesc' (optional).",
						Required:    false,
					},
					"due_after": {
						Type:        "string",
						Description: "Filter todos due after this date (ISO 8601 format, optional). Can be only used together with due_before.",
						Required:    false,
					},
					"due_before": {
						Type:        "string",
						Description: "Filter todos due before this date (ISO 8601 format, optional). Can be only used together with due_after.",
						Required:    false,
					},
				},
			},
		},
	}
}

// Call executes the TodoFetcherTool with the provided function call.
func (lft TodoFetcherTool) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall, _ []domain.LLMChatMessage) domain.LLMChatMessage {
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

	err := json.Unmarshal([]byte(call.Arguments), &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s"}`, err.Error()),
		}
	}

	opts := []domain.ListTodoOptions{}

	if params.SearchTerm != nil && *params.SearchTerm != "" {
		embedding, err := lft.llmCli.Embed(ctx, lft.llmEmbeddingModel, *params.SearchTerm)
		if err != nil {
			return domain.LLMChatMessage{
				Role:    domain.ChatRole_Tool,
				Content: fmt.Sprintf(`{"error":"embedding_error","details":"%s"}`, err.Error()),
			}
		}
		opts = append(opts, domain.WithEmbedding(embedding))
	}

	if params.Status != nil {
		opts = append(opts, domain.WithStatus(domain.TodoStatus(*params.Status)))
	}
	if params.SortBy != nil {
		opts = append(opts, domain.WithSortBy(*params.SortBy))
	}
	if params.DueAfter != nil && *params.DueAfter != "" {
		dueAfter, ok := domain.ExtractTimeFromText(*params.DueAfter, time.Now(), time.UTC)
		if !ok {
			return domain.LLMChatMessage{
				Role:    domain.ChatRole_Tool,
				Content: `{"error":"invalid_due_after","details":"Could not parse due_after date."}`,
			}
		}
		dueBefore, ok := domain.ExtractTimeFromText(*params.DueBefore, time.Now(), time.UTC)
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
			Description: "Create ONE new todo. REQUIRED keys: title (string), due_date ISO 8601 string. Do NOT include extra keys.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"title": {
						Type:        "string",
						Description: "Short task title. REQUIRED.",
						Required:    true,
					},
					"due_date": {
						Type:        "string",
						Description: "ISO 8601 date string. REQUIRED.",
						Required:    true,
					},
				},
			},
		},
	}
}

// Call executes the TodoCreatorTool with the provided function call.
func (tct TodoCreatorTool) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		Title   string `json:"title"`
		DueDate string `json:"due_date"`
	}{}

	err := json.Unmarshal([]byte(call.Arguments), &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s"}`, err.Error()),
		}
	}

	now := tct.timeProvider.Now()
	dueDate, found := extractDateParam(params.DueDate, conversationHistory, now)
	if !found {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: `{"error":"invalid_due_date","details":"Due date cannot be empty. ISO 8601 string is required."}`,
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
			Content: fmt.Sprintf(`{"error":"create_todo_error","details":"%s"}`, err.Error()),
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
			Description: "Update ONE existing todo. REQUIRED keys: id (UUID string) Optional: title, status. Do NOT include extra keys.",
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
func (tut TodoMetaUpdaterTool) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall, _ []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		ID     string  `json:"id"`
		Title  *string `json:"title"`
		Status *string `json:"status"`
	}{}

	err := json.Unmarshal([]byte(call.Arguments), &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s"}`, err.Error()),
		}
	}

	todoID, err := uuid.Parse(params.ID)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_todo_id","details":"%s"}`, err.Error()),
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
			Content: fmt.Sprintf(`{"error":"update_todo_error","details":"%s"}`, err.Error()),
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
			Description: "Update the due date of ONE existing todo. REQUIRED keys: id (UUID string) and due_date (ISO8601). Do NOT include extra keys.",
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
						Description: "ISO8601 date string. REQUIRED.",
						Required:    true,
					},
				},
			},
		},
	}
}

// Call executes the TodoDueDateUpdaterTool with the provided function call.
func (tdut TodoDueDateUpdaterTool) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		ID      uuid.UUID `json:"id"`
		DueDate string    `json:"due_date"`
	}{}

	err := json.Unmarshal([]byte(call.Arguments), &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s"}`, err.Error()),
		}
	}

	if params.ID == uuid.Nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: `{"error":"invalid_todo_id","details":"Todo ID cannot be nil."}`,
		}
	}

	now := tdut.timeProvider.Now()
	dueDate, found := extractDateParam(params.DueDate, conversationHistory, now)
	if !found {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: `{"error":"invalid_due_date","details":"Due date cannot be empty. ISO 8601 string is required."}`,
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
			Description: "Delete ONE todo by id (UUID). REQUIRED key: id. Do NOT include extra keys. Call fetch_todos first if you don't have the UUID.",
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
func (tdt TodoDeleterTool) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		ID uuid.UUID `json:"id"`
	}{}

	err := json.Unmarshal([]byte(call.Arguments), &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf(`{"error":"invalid_arguments","details":"%s"}`, err.Error()),
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
