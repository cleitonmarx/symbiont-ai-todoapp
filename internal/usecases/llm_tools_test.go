package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLLMToolManager(t *testing.T) {
	tests := map[string]struct {
		setupTools func() []domain.LLMTool
		setupMocks func(*domain.MockLLMTool)
		testFunc   func(t *testing.T, manager LLMToolManager)
	}{
		"list-returns-all-tools": {
			setupTools: func() []domain.LLMTool {
				tool1 := domain.NewMockLLMTool(t)
				tool1.EXPECT().
					Definition().
					Return(domain.LLMToolDefinition{
						Type: "function",
						Function: domain.LLMToolFunction{
							Name: "fetch_todos",
						},
					})

				tool2 := domain.NewMockLLMTool(t)
				tool2.EXPECT().
					Definition().
					Return(domain.LLMToolDefinition{
						Type: "function",
						Function: domain.LLMToolFunction{
							Name: "create_todo",
						},
					})

				return []domain.LLMTool{tool1, tool2}
			},
			testFunc: func(t *testing.T, manager LLMToolManager) {
				tools := manager.List()
				assert.Len(t, tools, 2)
				gotToolNames := []string{}
				for _, tool := range tools {
					gotToolNames = append(gotToolNames, tool.Function.Name)
				}

				assert.ElementsMatch(t, []string{"fetch_todos", "create_todo"}, gotToolNames)
			},
		},
		"status-message-returns-tool-specific-message": {
			setupTools: func() []domain.LLMTool {
				tool := domain.NewMockLLMTool(t)
				tool.EXPECT().
					Definition().
					Return(domain.LLMToolDefinition{
						Type: "function",
						Function: domain.LLMToolFunction{
							Name: "fetch_todos",
						},
					})
				tool.EXPECT().
					StatusMessage().
					Return("🔎 Fetching todos...\n\n")

				return []domain.LLMTool{tool}
			},
			testFunc: func(t *testing.T, manager LLMToolManager) {
				msg := manager.StatusMessage("fetch_todos")
				assert.Equal(t, "🔎 Fetching todos...\n\n", msg)
			},
		},
		"status-message-returns-default-when-tool-message-empty": {
			setupTools: func() []domain.LLMTool {
				tool := domain.NewMockLLMTool(t)
				tool.EXPECT().
					Definition().
					Return(domain.LLMToolDefinition{
						Type: "function",
						Function: domain.LLMToolFunction{
							Name: "fetch_todos",
						},
					})
				tool.EXPECT().
					StatusMessage().
					Return("")

				return []domain.LLMTool{tool}
			},
			testFunc: func(t *testing.T, manager LLMToolManager) {
				msg := manager.StatusMessage("fetch_todos")
				assert.Equal(t, "⏳ Processing request...\n\n", msg)
			},
		},
		"status-message-returns-default-when-tool-not-found": {
			setupTools: func() []domain.LLMTool {
				return []domain.LLMTool{}
			},
			testFunc: func(t *testing.T, manager LLMToolManager) {
				msg := manager.StatusMessage("unknown_tool")
				assert.Equal(t, "⏳ Processing request...\n\n", msg)
			},
		},
		"call-executes-correct-tool": {
			setupTools: func() []domain.LLMTool {
				tool := domain.NewMockLLMTool(t)
				tool.EXPECT().
					Definition().
					Return(domain.LLMToolDefinition{
						Type: "function",
						Function: domain.LLMToolFunction{
							Name: "fetch_todos",
						},
					})
				tool.EXPECT().
					Call(mock.Anything, mock.Anything, mock.Anything).
					Return(domain.LLMChatMessage{
						Role:    domain.ChatRole_Tool,
						Content: "todos found",
					})

				return []domain.LLMTool{tool}
			},
			testFunc: func(t *testing.T, manager LLMToolManager) {
				result := manager.Call(
					context.Background(),
					domain.LLMStreamEventFunctionCall{
						Function:  "fetch_todos",
						Arguments: "{}",
					},
					[]domain.LLMChatMessage{},
				)
				assert.Equal(t, domain.ChatRole_Tool, result.Role)
				assert.Equal(t, "todos found", result.Content)
			},
		},
		"call-returns-error-for-unknown-tool": {
			setupTools: func() []domain.LLMTool {
				return []domain.LLMTool{}
			},
			testFunc: func(t *testing.T, manager LLMToolManager) {
				result := manager.Call(
					context.Background(),
					domain.LLMStreamEventFunctionCall{
						Function: "unknown_tool",
					},
					[]domain.LLMChatMessage{},
				)
				assert.Equal(t, domain.ChatRole_Tool, result.Role)
				assert.Contains(t, result.Content, "unknown_tool")
				assert.Contains(t, result.Content, "not registered")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tools := tt.setupTools()
			manager := NewLLMToolManager(tools...)
			tt.testFunc(t, manager)
		})
	}
}

func TestTodoFetcherTool(t *testing.T) {
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setupMocks func(
			*domain.MockTodoRepository,
			*domain.MockLLMClient,
		)
		functionCall domain.LLMStreamEventFunctionCall
		validateResp func(t *testing.T, resp domain.LLMChatMessage)
	}{
		"fetch-todos-success": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Return([]domain.Todo{
						{
							ID:      uuid.New(),
							Title:   "Test Todo",
							DueDate: fixedTime,
							Status:  domain.TodoStatus_OPEN,
						},
					}, false, nil).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.NotNil(t, output["todos"])
			},
		},
		"fetch-todos-with-status-and-search": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
				llmCli.EXPECT().
					Embed(mock.Anything, "embedding-model", "urgent").
					Return([]float64{0.3, 0.4}, nil).
					Once()

				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...domain.ListTodoOptions) {
						param := domain.ListTodosParams{}
						for _, opt := range opts {
							opt(&param)
						}
						assert.Equal(t, domain.TodoStatus_OPEN, *param.Status)
						assert.Equal(t, []float64{0.3, 0.4}, param.Embedding)
					}).
					Return([]domain.Todo{
						{
							ID:      uuid.New(),
							Title:   "Urgent Todo",
							DueDate: fixedTime,
							Status:  domain.TodoStatus_OPEN,
						},
					}, false, nil).
					Once()

			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10, "status": "OPEN", "search_term": "urgent"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.NotNil(t, output["todos"])
			},
		},
		"fetch-todos-with-sortby": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...domain.ListTodoOptions) {
						param := domain.ListTodosParams{}
						for _, opt := range opts {
							opt(&param)
						}
						assert.Equal(t, &domain.TodoSortBy{Field: "duedate", Direction: "ASC"}, param.SortBy)
					}).
					Return([]domain.Todo{}, false, nil)
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10, "sort_by": "duedateAsc"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.Nil(t, output["todos"])
			},
		},
		"fetch-todos-with-due-date-filters": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...domain.ListTodoOptions) {
						param := domain.ListTodosParams{}
						for _, opt := range opts {
							opt(&param)
						}
						expectedDueAfter, _ := time.Parse("2006-01-02", "2026-01-20")
						expectedDueBefore, _ := time.Parse("2006-01-02", "2026-01-30")
						assert.Equal(t, expectedDueAfter, *param.DueAfter)
						assert.Equal(t, expectedDueBefore, *param.DueBefore)
					}).
					Return([]domain.Todo{
						{
							ID:      uuid.New(),
							Title:   "Urgent Todo",
							DueDate: fixedTime,
							Status:  domain.TodoStatus_OPEN,
						},
					}, false, nil)

			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10, "due_after": "2026-01-20", "due_before": "2026-01-30"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.NotNil(t, output["todos"])
			},
		},
		"fetch-todos-invalid-due-after": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10, "due_after": "invalid-date"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_after")
			},
		},
		"fetch-todos-invalid-due-before": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10, "due_after": "2026-01-20", "due_before": "invalid-date"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_before")
			},
		},

		"fetch-todos-invalid-arguments": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `invalid json`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"fetch-todos-embedding-error": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
				llmCli.EXPECT().
					Embed(mock.Anything, "embedding-model", "search").
					Return(nil, errors.New("embedding failed")).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10, "search_term": "search"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "embedding_error")
			},
		},
		"fetch-todos-list-error": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
				llmCli.EXPECT().
					Embed(mock.Anything, "embedding-model", "search").
					Return([]float64{0.1, 0.2}, nil).
					Once()

				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 10, mock.Anything).
					Return(nil, false, errors.New("db error")).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10, "search_term": "search"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "list_todos_error")
			},
		},
		"fetch-todos-has-more": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
				llmCli.EXPECT().
					Embed(mock.Anything, "embedding-model", "search").
					Return([]float64{0.1, 0.2}, nil).
					Once()

				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 10, mock.Anything).
					Return([]domain.Todo{
						{
							ID:      uuid.New(),
							Title:   "Test Todo",
							DueDate: fixedTime,
							Status:  domain.TodoStatus_OPEN,
						},
					}, true, nil).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10, "search_term": "search"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.Equal(t, float64(2), output["next_page"])
			},
		},
		"fetch-todos-no-results": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, llmCli *domain.MockLLMClient) {
				llmCli.EXPECT().
					Embed(mock.Anything, "embedding-model", "search").
					Return([]float64{0.1, 0.2}, nil).
					Once()

				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 10, mock.Anything).
					Return([]domain.Todo{}, false, nil).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "fetch_todos",
				Arguments: `{"page": 1, "page_size": 10, "search_term": "search"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "no_todos_found")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			todoRepo := domain.NewMockTodoRepository(t)
			llmCli := domain.NewMockLLMClient(t)
			tt.setupMocks(todoRepo, llmCli)

			tool := NewTodoFetcherTool(todoRepo, llmCli, "embedding-model")

			resp := tool.Call(context.Background(), tt.functionCall, []domain.LLMChatMessage{})
			tt.validateResp(t, resp)
		})
	}
}

func TestTodoCreatorTool(t *testing.T) {
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setupMocks func(
			*domain.MockUnitOfWork,
			*domain.MockCurrentTimeProvider,
			*MockTodoCreator,
		)
		functionCall domain.LLMStreamEventFunctionCall
		history      []domain.LLMChatMessage
		validateResp func(t *testing.T, resp domain.LLMChatMessage)
	}{
		"create-todo-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				creator.EXPECT().
					Create(mock.Anything, uow, "New Todo", mock.Anything).
					Return(domain.Todo{Title: "New Todo"}, nil).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": "2026-01-25"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "created successfully")
			},
		},
		"create-todo-empty-due-date-uses-history": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				creator.EXPECT().
					Create(mock.Anything, uow, "New Todo", mock.Anything).
					Return(domain.Todo{Title: "New Todo"}, nil).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": ""}`,
			},
			history: []domain.LLMChatMessage{
				{Role: domain.ChatRole_User, Content: "Please set it for tomorrow"},
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "created successfully")
			},
		},
		"create-todo-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "create_todo",
				Arguments: `invalid json`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"create-todo-invalid-due-date": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": "invalid"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_date")
			},
		},
		"create-todo-create-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				creator.EXPECT().
					Create(mock.Anything, uow, "New Todo", mock.Anything).
					Return(domain.Todo{}, errors.New("create error")).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": "2026-01-25"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "create_todo_error")
			},
		},
		"create-todo-uow-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, creator *MockTodoCreator) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					Return(errors.New("uow error")).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "create_todo",
				Arguments: `{"title": "New Todo", "due_date": "2026-01-25"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "create_todo_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			todoCreator := NewMockTodoCreator(t)
			tt.setupMocks(uow, timeProvider, todoCreator)

			tool := NewTodoCreatorTool(uow, todoCreator, timeProvider)

			resp := tool.Call(context.Background(), tt.functionCall, tt.history)
			tt.validateResp(t, resp)
		})
	}
}

func TestTodoMetaUpdaterTool(t *testing.T) {
	todoID := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*domain.MockUnitOfWork, *MockTodoUpdater)
		functionCall domain.LLMStreamEventFunctionCall
		validateResp func(t *testing.T, resp domain.LLMChatMessage)
	}{
		"update-todo-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, updater *MockTodoUpdater) {
				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						common.Ptr("Updated"),
						common.Ptr(domain.TodoStatus_DONE),
						(*time.Time)(nil),
					).
					Return(
						domain.Todo{
							ID:     todoID,
							Title:  "Updated",
							Status: domain.TodoStatus_DONE,
						},
						nil,
					)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "update_todo",
				Arguments: `{"id": "` + todoID.String() + `", "title": "Updated", "status": "DONE"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "updated successfully")
			},
		},
		"update-todo-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, updater *MockTodoUpdater) {
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "update_todo",
				Arguments: `invalid json`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"update-todo-invalid-id": {
			setupMocks: func(uow *domain.MockUnitOfWork, updater *MockTodoUpdater) {
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "update_todo",
				Arguments: `{"id": "invalid-uuid", "title": "Updated"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_todo_id")
			},
		},
		"update-todo-update-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, updater *MockTodoUpdater) {
				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						common.Ptr("Updated"),
						(*domain.TodoStatus)(nil),
						(*time.Time)(nil),
					).
					Return(domain.Todo{}, errors.New("update error"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "update_todo",
				Arguments: `{"id": "` + todoID.String() + `", "title": "Updated"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "update_todo_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			updater := NewMockTodoUpdater(t)
			tt.setupMocks(uow, updater)

			tool := NewTodoMetaUpdaterTool(uow, updater)

			resp := tool.Call(context.Background(), tt.functionCall, []domain.LLMChatMessage{})
			tt.validateResp(t, resp)
		})
	}
}

func TestTodoDueDateUpdaterTool(t *testing.T) {
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)
	todoID := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*domain.MockUnitOfWork, *domain.MockCurrentTimeProvider, *MockTodoUpdater)
		functionCall domain.LLMStreamEventFunctionCall
		history      []domain.LLMChatMessage
		validateResp func(t *testing.T, resp domain.LLMChatMessage)
	}{
		"update-due-date-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						(*string)(nil),
						(*domain.TodoStatus)(nil),
						common.Ptr(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
					).
					Return(
						domain.Todo{
							ID:      todoID,
							Title:   "Some Todo",
							DueDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
						},
						nil,
					)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "update_todo_due_date",
				Arguments: `{"id": "` + todoID.String() + `", "due_date": "2026-02-01"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "updated successfully")
			},
		},
		"update-due-date-uses-history": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						(*string)(nil),
						(*domain.TodoStatus)(nil),
						mock.Anything,
					).
					Return(
						domain.Todo{
							ID:    todoID,
							Title: "Some Todo",
						},
						nil,
					)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "update_todo_due_date",
				Arguments: `{"id": "` + todoID.String() + `", "due_date": ""}`,
			},
			history: []domain.LLMChatMessage{
				{Role: domain.ChatRole_User, Content: "Please set it to tomorrow"},
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "updated successfully")
			},
		},
		"update-due-date-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "update_todo_due_date",
				Arguments: `invalid json`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"update-due-date-invalid-id": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "update_todo_due_date",
				Arguments: `{"id": "00000000-0000-0000-0000-000000000000", "due_date": "2026-02-01"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_todo_id")
			},
		},
		"update-due-date-update-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider, updater *MockTodoUpdater) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				updater.EXPECT().
					Update(
						mock.Anything,
						uow,
						todoID,
						(*string)(nil),
						(*domain.TodoStatus)(nil),
						mock.Anything,
					).
					Return(domain.Todo{}, errors.New("update error"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "update_todo_due_date",
				Arguments: `{"id": "` + todoID.String() + `", "due_date": "2026-02-01"}`,
			},
			history: []domain.LLMChatMessage{},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "update_due_date_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			updater := NewMockTodoUpdater(t)
			tt.setupMocks(uow, timeProvider, updater)

			tool := NewTodoDueDateUpdaterTool(uow, updater, timeProvider)

			resp := tool.Call(context.Background(), tt.functionCall, tt.history)
			tt.validateResp(t, resp)
		})
	}
}

func TestTodoDeleterTool(t *testing.T) {
	todoID := uuid.New()

	tests := map[string]struct {
		setupMocks   func(*domain.MockUnitOfWork, *MockTodoDeleter)
		functionCall domain.LLMStreamEventFunctionCall
		validateResp func(t *testing.T, resp domain.LLMChatMessage)
	}{
		"delete-todo-success": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *MockTodoDeleter) {
				deleter.EXPECT().
					Delete(
						mock.Anything,
						uow,
						todoID,
					).
					Return(nil)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "delete_todo",
				Arguments: `{"id": "` + todoID.String() + `"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "deleted successfully")
			},
		},
		"delete-todo-invalid-arguments": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *MockTodoDeleter) {
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "delete_todo",
				Arguments: `invalid json`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"delete-todo-delete-error": {
			setupMocks: func(uow *domain.MockUnitOfWork, deleter *MockTodoDeleter) {
				deleter.EXPECT().
					Delete(
						mock.Anything,
						uow,
						todoID,
					).
					Return(errors.New("delete error"))

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
			functionCall: domain.LLMStreamEventFunctionCall{
				Function:  "delete_todo",
				Arguments: `{"id": "` + todoID.String() + `"}`,
			},
			validateResp: func(t *testing.T, resp domain.LLMChatMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "delete_todo_error")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			deleter := NewMockTodoDeleter(t)
			tt.setupMocks(uow, deleter)

			tool := NewTodoDeleterTool(uow, deleter)

			resp := tool.Call(context.Background(), tt.functionCall, []domain.LLMChatMessage{})
			tt.validateResp(t, resp)
		})
	}
}

func TestLLMToolManager_List_And_StatusMessages(t *testing.T) {
	manager := NewLLMToolManager(
		NewTodoFetcherTool(nil, nil, ""),
		NewTodoCreatorTool(nil, nil, nil),
		NewTodoMetaUpdaterTool(nil, nil),
		NewTodoDueDateUpdaterTool(nil, nil, nil),
		NewTodoDeleterTool(nil, nil),
	)

	tools := manager.List()
	require.Len(t, tools, 5)

	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Function.Name)
	}

	assert.ElementsMatch(t, []string{
		"fetch_todos",
		"create_todo",
		"update_todo",
		"update_todo_due_date",
		"delete_todo",
	}, names)

	statusMessages := []string{}
	for _, name := range names {
		msg := manager.StatusMessage(name)
		statusMessages = append(statusMessages, msg)
	}

	assert.ElementsMatch(t, []string{
		"🔎 Fetching todos...\n\n",
		"📝 Creating your todo...\n\n",
		"✏️ Updating your todo...\n\n",
		"📅 Updating the due date...\n\n",
		"🗑️ Deleting the todo...\n\n",
	}, statusMessages)
}

func TestInitLLMToolRegistry_Initialize(t *testing.T) {
	i := InitLLMToolRegistry{}

	ctx, err := i.Initialize(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, ctx)

	r, err := depend.Resolve[domain.LLMToolRegistry]()
	require.NoError(t, err)
	assert.NotNil(t, r)

}
