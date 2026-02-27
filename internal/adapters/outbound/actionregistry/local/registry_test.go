package local

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestActionRegistry(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupActions   func() []domain.AssistantAction
		embeddingModel string
		testFunc       func(t *testing.T, manager LocalRegistry)
	}{
		"status-message-returns-action-specific-message": {
			setupActions: func() []domain.AssistantAction {
				action := domain.NewMockAssistantAction(t)
				action.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos"))
				action.EXPECT().StatusMessage().Return("🔎 Fetching todos...")
				return []domain.AssistantAction{action}
			},
			testFunc: func(t *testing.T, manager LocalRegistry) {
				msg := manager.StatusMessage("fetch_todos")
				assert.Equal(t, "🔎 Fetching todos...", msg)
			},
		},
		"status-message-returns-default-when-action-not-found": {
			setupActions: func() []domain.AssistantAction { return []domain.AssistantAction{} },
			testFunc: func(t *testing.T, manager LocalRegistry) {
				msg := manager.StatusMessage("unknown_action")
				assert.Equal(t, "⏳ Processing request...", msg)
			},
		},
		"execute-calls-correct-action": {
			setupActions: func() []domain.AssistantAction {
				action := domain.NewMockAssistantAction(t)
				action.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos"))
				action.EXPECT().
					Execute(mock.Anything, mock.MatchedBy(func(call domain.AssistantActionCall) bool {
						return call.Name == "fetch_todos" && call.Input == "{}" && call.ID == "call-1"
					}), mock.MatchedBy(func(history []domain.AssistantMessage) bool {
						return len(history) == 1 && history[0].Role == domain.ChatRole_User && history[0].Content == "hi"
					})).
					Return(domain.AssistantMessage{
						Role:         domain.ChatRole_Tool,
						Content:      "todos found",
						ActionCallID: common.Ptr("call-1"),
					})
				return []domain.AssistantAction{action}
			},
			testFunc: func(t *testing.T, manager LocalRegistry) {
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
		"execute-returns-error-for-unknown-action": {
			setupActions: func() []domain.AssistantAction { return []domain.AssistantAction{} },
			testFunc: func(t *testing.T, manager LocalRegistry) {
				result := manager.Execute(
					context.Background(),
					domain.AssistantActionCall{ID: "x", Name: "unknown_action", Input: ""},
					nil,
				)
				assert.Equal(t, domain.ChatRole_Tool, result.Role)
				assert.Contains(t, result.Content, "unknown_action")
				assert.Contains(t, result.Content, "not registered")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager := NewActionRegistry(nil, "", tt.setupActions()...)
			tt.testFunc(t, manager)
		})
	}
}

func TestActionRegistry_GetDefinition(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		actionName   string
		setupActions func(t *testing.T) []domain.AssistantAction
		assertResult func(t *testing.T, definition domain.AssistantActionDefinition, found bool)
	}{
		"returns-definition-when-action-exists": {
			actionName: "fetch_todos",
			setupActions: func(t *testing.T) []domain.AssistantAction {
				action := domain.NewMockAssistantAction(t)
				action.EXPECT().
					Definition().
					Return(domain.AssistantActionDefinition{
						Name:        "fetch_todos",
						Description: "Fetch todos from storage",
						Input: domain.AssistantActionInput{
							Type: "object",
						},
					}).
					Maybe()
				return []domain.AssistantAction{action}
			},
			assertResult: func(t *testing.T, definition domain.AssistantActionDefinition, found bool) {
				assert.True(t, found)
				assert.Equal(t, "fetch_todos", definition.Name)
				assert.Equal(t, "Fetch todos from storage", definition.Description)
				assert.Equal(t, "object", definition.Input.Type)
			},
		},
		"returns-not-found-when-action-does-not-exist": {
			actionName: "missing_action",
			setupActions: func(t *testing.T) []domain.AssistantAction {
				action := domain.NewMockAssistantAction(t)
				action.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos")).
					Maybe()
				return []domain.AssistantAction{action}
			},
			assertResult: func(t *testing.T, definition domain.AssistantActionDefinition, found bool) {
				assert.False(t, found)
				assert.Equal(t, domain.AssistantActionDefinition{}, definition)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registry := NewActionRegistry(nil, "", tt.setupActions(t)...)
			definition, found := registry.GetDefinition(tt.actionName)
			tt.assertResult(t, definition, found)
		})
	}
}

func TestInitActionRegistry_Initialize(t *testing.T) {
	t.Parallel()

	i := InitLocalActionRegistry{}
	registry, err := i.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, registry)

	dependency, err := depend.ResolveNamed[domain.AssistantActionRegistry]("local")
	assert.NoError(t, err)
	assert.IsType(t, LocalRegistry{}, dependency)
}

func mockAssistantActionDefinition(name string) domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name: name,
		Input: domain.AssistantActionInput{
			Type:   "object",
			Fields: map[string]domain.AssistantActionField{},
		},
	}
}
