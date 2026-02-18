package assistant

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/assistant/actions"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAssistantActionManager(t *testing.T) {
	tests := map[string]struct {
		setupActions func() []domain.AssistantAction
		testFunc     func(t *testing.T, manager AssistantActionManager)
	}{
		"list-returns-all-actions": {
			setupActions: func() []domain.AssistantAction {
				tool1 := domain.NewMockAssistantAction(t)
				tool1.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos"))

				tool2 := domain.NewMockAssistantAction(t)
				tool2.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("create_todo"))

				return []domain.AssistantAction{tool1, tool2}
			},
			testFunc: func(t *testing.T, manager AssistantActionManager) {
				actions := manager.List()
				assert.Len(t, actions, 2)
				names := []string{}
				for _, action := range actions {
					names = append(names, action.Name)
				}
				assert.ElementsMatch(t, []string{"fetch_todos", "create_todo"}, names)
			},
		},
		"status-message-returns-tool-specific-message": {
			setupActions: func() []domain.AssistantAction {
				tool := domain.NewMockAssistantAction(t)
				tool.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos"))
				tool.EXPECT().StatusMessage().Return("🔎 Fetching todos...")
				return []domain.AssistantAction{tool}
			},
			testFunc: func(t *testing.T, manager AssistantActionManager) {
				msg := manager.StatusMessage("fetch_todos")
				assert.Equal(t, "🔎 Fetching todos...", msg)
			},
		},
		"status-message-returns-default-when-tool-not-found": {
			setupActions: func() []domain.AssistantAction { return []domain.AssistantAction{} },
			testFunc: func(t *testing.T, manager AssistantActionManager) {
				msg := manager.StatusMessage("unknown_tool")
				assert.Equal(t, "⏳ Processing request...", msg)
			},
		},
		"execute-calls-correct-tool": {
			setupActions: func() []domain.AssistantAction {
				tool := domain.NewMockAssistantAction(t)
				tool.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos"))
				tool.EXPECT().
					Execute(mock.Anything, mock.MatchedBy(func(call domain.AssistantActionCall) bool {
						return call.Name == "fetch_todos" && call.Input == "{}" && call.ID == "call-1"
					}), mock.MatchedBy(func(history []domain.AssistantMessage) bool {
						return len(history) == 1 && history[0].Role == domain.ChatRole_User && history[0].Content == "hi"
					})).
					Return(domain.AssistantMessage{
						Role:         domain.ChatRole_Tool,
						Content:      "todos found",
						ActionCallID: ptr("call-1"),
					})
				return []domain.AssistantAction{tool}
			},
			testFunc: func(t *testing.T, manager AssistantActionManager) {
				result := manager.Execute(
					context.Background(),
					domain.AssistantActionCall{
						ID:    "call-1",
						Name:  "fetch_todos",
						Input: "{}",
					},
					[]domain.AssistantMessage{{Role: domain.ChatRole_User, Content: "hi"}},
				)
				assert.Equal(t, domain.ChatRole_Tool, result.Role)
				assert.Equal(t, "todos found", result.Content)
				if assert.NotNil(t, result.ActionCallID) {
					assert.Equal(t, "call-1", *result.ActionCallID)
				}
			},
		},
		"execute-returns-error-for-unknown-tool": {
			setupActions: func() []domain.AssistantAction { return []domain.AssistantAction{} },
			testFunc: func(t *testing.T, manager AssistantActionManager) {
				result := manager.Execute(
					context.Background(),
					domain.AssistantActionCall{ID: "x", Name: "unknown_tool"},
					nil,
				)
				assert.Equal(t, domain.ChatRole_Tool, result.Role)
				assert.Contains(t, result.Content, "unknown_tool")
				assert.Contains(t, result.Content, "not registered")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager := NewAssistantActionManager(tt.setupActions()...)
			tt.testFunc(t, manager)
		})
	}
}

func TestAssistantActionManager_List_And_StatusMessages(t *testing.T) {
	manager := NewAssistantActionManager(
		actions.NewUIFiltersSetterAction(),
		actions.NewTodoFetcherAction(nil, nil, nil, ""),
		actions.NewTodoCreatorAction(nil, nil, nil),
		actions.NewTodoUpdaterAction(nil, nil),
		actions.NewTodoDueDateUpdaterAction(nil, nil, nil),
		actions.NewTodoDeleterAction(nil, nil),
	)

	actions := manager.List()
	require.Len(t, actions, 6)

	names := make([]string, 0, len(actions))
	for _, action := range actions {
		names = append(names, action.Name)
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
		statusMessages = append(statusMessages, manager.StatusMessage(name))
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

func TestInitAssistantActionRegistry_Initialize(t *testing.T) {
	i := InitAssistantActionRegistry{}

	ctx, err := i.Initialize(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, ctx)

	r, err := depend.Resolve[domain.AssistantActionRegistry]()
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func ptr[T any](v T) *T { return &v }

func mockAssistantActionDefinition(name string) domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name: name,
		Input: domain.AssistantActionInput{
			Type:   "object",
			Fields: map[string]domain.AssistantActionField{},
		},
	}
}
