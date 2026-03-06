package local

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestActionRegistry(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupActions   func() []assistant.Action
		embeddingModel string
		testFunc       func(t *testing.T, manager LocalRegistry)
	}{
		"status-message-returns-action-specific-message": {
			setupActions: func() []assistant.Action {
				action := assistant.NewMockAction(t)
				action.EXPECT().
					Definition().
					Return(mockActionDefinition("fetch_todos"))
				action.EXPECT().StatusMessage().Return("🔎 Fetching todos...")
				return []assistant.Action{action}
			},
			testFunc: func(t *testing.T, manager LocalRegistry) {
				msg := manager.StatusMessage("fetch_todos")
				assert.Equal(t, "🔎 Fetching todos...", msg)
			},
		},
		"status-message-returns-default-when-action-not-found": {
			setupActions: func() []assistant.Action { return []assistant.Action{} },
			testFunc: func(t *testing.T, manager LocalRegistry) {
				msg := manager.StatusMessage("unknown_action")
				assert.Equal(t, "⏳ Processing request...", msg)
			},
		},
		"execute-calls-correct-action": {
			setupActions: func() []assistant.Action {
				action := assistant.NewMockAction(t)
				action.EXPECT().
					Definition().
					Return(mockActionDefinition("fetch_todos"))
				action.EXPECT().
					Execute(mock.Anything, mock.MatchedBy(func(call assistant.ActionCall) bool {
						return call.Name == "fetch_todos" && call.Input == "{}" && call.ID == "call-1"
					}), mock.MatchedBy(func(history []assistant.Message) bool {
						return len(history) == 1 && history[0].Role == assistant.ChatRole_User && history[0].Content == "hi"
					})).
					Return(assistant.Message{
						Role:         assistant.ChatRole_Tool,
						Content:      "todos found",
						ActionCallID: common.Ptr("call-1"),
					})
				return []assistant.Action{action}
			},
			testFunc: func(t *testing.T, manager LocalRegistry) {
				result := manager.Execute(
					context.Background(),
					assistant.ActionCall{
						ID:    "call-1",
						Name:  "fetch_todos",
						Input: "{}",
					},
					[]assistant.Message{{Role: assistant.ChatRole_User, Content: "hi"}},
				)
				assert.Equal(t, assistant.ChatRole_Tool, result.Role)
				assert.Equal(t, "todos found", result.Content)
				if assert.NotNil(t, result.ActionCallID) {
					assert.Equal(t, "call-1", *result.ActionCallID)
				}
			},
		},
		"execute-returns-error-for-unknown-action": {
			setupActions: func() []assistant.Action { return []assistant.Action{} },
			testFunc: func(t *testing.T, manager LocalRegistry) {
				result := manager.Execute(
					context.Background(),
					assistant.ActionCall{ID: "x", Name: "unknown_action", Input: ""},
					nil,
				)
				assert.Equal(t, assistant.ChatRole_Tool, result.Role)
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
		setupActions func(t *testing.T) []assistant.Action
		assertResult func(t *testing.T, definition assistant.ActionDefinition, found bool)
	}{
		"returns-definition-when-action-exists": {
			actionName: "fetch_todos",
			setupActions: func(t *testing.T) []assistant.Action {
				action := assistant.NewMockAction(t)
				action.EXPECT().
					Definition().
					Return(assistant.ActionDefinition{
						Name:        "fetch_todos",
						Description: "Fetch todos from storage",
						Input: assistant.ActionInput{
							Type: "object",
						},
					}).
					Maybe()
				return []assistant.Action{action}
			},
			assertResult: func(t *testing.T, definition assistant.ActionDefinition, found bool) {
				assert.True(t, found)
				assert.Equal(t, "fetch_todos", definition.Name)
				assert.Equal(t, "Fetch todos from storage", definition.Description)
				assert.Equal(t, "object", definition.Input.Type)
			},
		},
		"returns-not-found-when-action-does-not-exist": {
			actionName: "missing_action",
			setupActions: func(t *testing.T) []assistant.Action {
				action := assistant.NewMockAction(t)
				action.EXPECT().
					Definition().
					Return(mockActionDefinition("fetch_todos")).
					Maybe()
				return []assistant.Action{action}
			},
			assertResult: func(t *testing.T, definition assistant.ActionDefinition, found bool) {
				assert.False(t, found)
				assert.Equal(t, assistant.ActionDefinition{}, definition)
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

func TestActionRegistry_GetRenderer(t *testing.T) {
	t.Parallel()

	renderer := &actionsMockRenderer{}

	tests := map[string]struct {
		actionName   string
		setupActions func(t *testing.T) []assistant.Action
		assertResult func(t *testing.T, got assistant.ActionResultRenderer, found bool)
	}{
		"returns-renderer-when-action-exists": {
			actionName: "fetch_todos",
			setupActions: func(t *testing.T) []assistant.Action {
				action := assistant.NewMockAction(t)
				action.EXPECT().Definition().Return(mockActionDefinition("fetch_todos")).Maybe()
				action.EXPECT().Renderer().Return(renderer, true).Once()
				return []assistant.Action{action}
			},
			assertResult: func(t *testing.T, got assistant.ActionResultRenderer, found bool) {
				assert.True(t, found)
				assert.Same(t, renderer, got)
			},
		},
		"returns-not-found-when-action-does-not-exist": {
			actionName:   "missing_action",
			setupActions: func(t *testing.T) []assistant.Action { return nil },
			assertResult: func(t *testing.T, got assistant.ActionResultRenderer, found bool) {
				assert.False(t, found)
				assert.Nil(t, got)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registry := NewActionRegistry(nil, "", tt.setupActions(t)...)
			got, found := registry.GetRenderer(tt.actionName)
			tt.assertResult(t, got, found)
		})
	}
}

func mockActionDefinition(name string) assistant.ActionDefinition {
	return assistant.ActionDefinition{
		Name: name,
		Input: assistant.ActionInput{
			Type:   "object",
			Fields: map[string]assistant.ActionField{},
		},
	}
}

type actionsMockRenderer struct{}

func (actionsMockRenderer) Render(_ assistant.ActionCall, _ assistant.Message) (assistant.Message, bool) {
	return assistant.Message{}, false
}
