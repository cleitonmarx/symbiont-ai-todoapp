package usecases

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
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
					Return("🔎 Fetching todos...")

				return []domain.LLMTool{tool}
			},
			testFunc: func(t *testing.T, manager LLMToolManager) {
				msg := manager.StatusMessage("fetch_todos")
				assert.Equal(t, "🔎 Fetching todos...", msg)
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
				assert.Equal(t, "⏳ Processing request...", msg)
			},
		},
		"status-message-returns-default-when-tool-not-found": {
			setupTools: func() []domain.LLMTool {
				return []domain.LLMTool{}
			},
			testFunc: func(t *testing.T, manager LLMToolManager) {
				msg := manager.StatusMessage("unknown_tool")
				assert.Equal(t, "⏳ Processing request...", msg)
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
					domain.LLMStreamEventToolCall{
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
					domain.LLMStreamEventToolCall{
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

func TestLLMToolManager_List_And_StatusMessages(t *testing.T) {
	manager := NewLLMToolManager(
		NewUIFiltersSetterTool(),
		NewTodoFetcherTool(nil, nil, nil, ""),
		NewTodoCreatorTool(nil, nil, nil),
		NewTodoUpdaterTool(nil, nil),
		NewTodoDueDateUpdaterTool(nil, nil, nil),
		NewTodoDeleterTool(nil, nil),
	)

	tools := manager.List()
	require.Len(t, tools, 6)

	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Function.Name)
	}

	assert.ElementsMatch(t, []string{
		"set_ui_filters",
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
		"🎛️ Applying filters...",
		"🔎 Fetching todos...",
		"📝 Creating your todo...",
		"✏️ Updating your todo...",
		"📅 Updating the due date...",
		"🗑️ Deleting the todo...",
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
