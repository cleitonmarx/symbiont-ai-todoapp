package composite

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCompositeActionRegistry_Execute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		call          assistant.ActionCall
		history       []assistant.Message
		registriesLen int
		setMocks      func(t *testing.T, registries []*assistant.MockActionRegistry)
		assertMessage func(t *testing.T, message assistant.Message)
	}{
		"routes-execution-to-matching-action": {
			call:          assistant.ActionCall{ID: "call-1", Name: "fetch_todos", Input: "{}"},
			history:       []assistant.Message{{Role: assistant.ChatRole_User, Content: "hello"}},
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*assistant.MockActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("fetch_todos").
					Return(assistant.ActionDefinition{Name: "fetch_todos"}, true).
					Once()
				registries[0].EXPECT().
					Execute(
						mock.Anything,
						assistant.ActionCall{ID: "call-1", Name: "fetch_todos", Input: "{}"},
						[]assistant.Message{{Role: assistant.ChatRole_User, Content: "hello"}},
					).
					Return(assistant.Message{
						Role:         assistant.ChatRole_Tool,
						Content:      "todos",
						ActionCallID: common.Ptr("call-1"),
					}).
					Once()
			},
			assertMessage: func(t *testing.T, message assistant.Message) {
				require.NotNil(t, message.ActionCallID)
				assert.Equal(t, assistant.ChatRole_Tool, message.Role)
				assert.Equal(t, "todos", message.Content)
				assert.Equal(t, "call-1", *message.ActionCallID)
			},
		},
		"returns-error-when-action-is-not-registered": {
			call:          assistant.ActionCall{ID: "call-2", Name: "unknown_action", Input: "{}"},
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*assistant.MockActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("unknown_action").
					Return(assistant.ActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("unknown_action").
					Return(assistant.ActionDefinition{}, false).
					Once()
			},
			assertMessage: func(t *testing.T, message assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, message.Role)
				assert.Equal(t, "error: no registry found for action 'unknown_action'", message.Content)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registryMocks := newMockRegistries(t, tt.registriesLen)
			if tt.setMocks != nil {
				tt.setMocks(t, registryMocks)
			}
			registries := make([]assistant.ActionRegistry, 0, tt.registriesLen)
			for _, mock := range registryMocks {
				registries = append(registries, mock)
			}

			registry := NewCompositeActionRegistry(t.Context(), registries...)
			message := registry.Execute(t.Context(), tt.call, tt.history)
			if tt.assertMessage != nil {
				tt.assertMessage(t, message)
			}
		})
	}
}

func TestCompositeActionRegistry_GetDefinition(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		actionName    string
		registriesLen int
		setMocks      func(t *testing.T, registries []*assistant.MockActionRegistry)
		assertResult  func(t *testing.T, definition assistant.ActionDefinition, found bool)
	}{
		"returns-definition-from-first-registry": {
			actionName:    "fetch_todos",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*assistant.MockActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("fetch_todos").
					Return(assistant.ActionDefinition{Name: "fetch_todos", Description: "from first"}, true).
					Once()
			},
			assertResult: func(t *testing.T, definition assistant.ActionDefinition, found bool) {
				assert.True(t, found)
				assert.Equal(t, "fetch_todos", definition.Name)
				assert.Equal(t, "from first", definition.Description)
			},
		},
		"falls-back-to-next-registry-when-first-misses": {
			actionName:    "update_todos",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*assistant.MockActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("update_todos").
					Return(assistant.ActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("update_todos").
					Return(assistant.ActionDefinition{Name: "update_todos", Description: "from second"}, true).
					Once()
			},
			assertResult: func(t *testing.T, definition assistant.ActionDefinition, found bool) {
				assert.True(t, found)
				assert.Equal(t, "update_todos", definition.Name)
				assert.Equal(t, "from second", definition.Description)
			},
		},
		"returns-not-found-when-missing-in-all-registries": {
			actionName:    "missing_action",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*assistant.MockActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("missing_action").
					Return(assistant.ActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("missing_action").
					Return(assistant.ActionDefinition{}, false).
					Once()
			},
			assertResult: func(t *testing.T, definition assistant.ActionDefinition, found bool) {
				assert.False(t, found)
				assert.Equal(t, assistant.ActionDefinition{}, definition)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registryMocks := newMockRegistries(t, tt.registriesLen)
			if tt.setMocks != nil {
				tt.setMocks(t, registryMocks)
			}

			registries := make([]assistant.ActionRegistry, 0, tt.registriesLen)
			for _, mockRegistry := range registryMocks {
				registries = append(registries, mockRegistry)
			}

			registry := NewCompositeActionRegistry(t.Context(), registries...)
			definition, found := registry.GetDefinition(tt.actionName)
			if tt.assertResult != nil {
				tt.assertResult(t, definition, found)
			}
		})
	}
}

func TestCompositeActionRegistry_StatusMessage(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		actionName    string
		registriesLen int
		setMocks      func(t *testing.T, registries []*assistant.MockActionRegistry)
		expected      string
	}{
		"returns-action-status-message-when-action-exists": {
			actionName:    "search_web",
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*assistant.MockActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("search_web").
					Return(assistant.ActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("search_web").
					Return(assistant.ActionDefinition{Name: "search_web"}, true).
					Once()
				registries[1].EXPECT().
					StatusMessage("search_web").
					Return("🔎 searching...").
					Once()
			},
			expected: "🔎 searching...",
		},
		"returns-default-status-message-when-action-does-not-exist": {
			actionName:    "missing_action",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*assistant.MockActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("missing_action").
					Return(assistant.ActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("missing_action").
					Return(assistant.ActionDefinition{}, false).
					Once()
			},
			expected: "⏳ Processing request...",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registryMocks := newMockRegistries(t, tt.registriesLen)
			if tt.setMocks != nil {
				tt.setMocks(t, registryMocks)
			}

			registries := make([]assistant.ActionRegistry, 0, tt.registriesLen)
			for _, mock := range registryMocks {
				registries = append(registries, mock)
			}

			registry := NewCompositeActionRegistry(t.Context(), registries...)
			assert.Equal(t, tt.expected, registry.StatusMessage(tt.actionName))
		})
	}
}

func TestCompositeActionRegistry_GetRenderer(t *testing.T) {
	t.Parallel()

	renderer := &compositeMockRenderer{}

	tests := map[string]struct {
		actionName    string
		registriesLen int
		setMocks      func(t *testing.T, registries []*assistant.MockActionRegistry)
		assertResult  func(t *testing.T, got assistant.ActionResultRenderer, found bool)
	}{
		"returns-renderer-from-first-registry-that-has-it": {
			actionName:    "fetch_todos",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*assistant.MockActionRegistry) {
				registries[0].EXPECT().
					GetRenderer("fetch_todos").
					Return(nil, false).
					Once()
				registries[1].EXPECT().
					GetRenderer("fetch_todos").
					Return(renderer, true).
					Once()
			},
			assertResult: func(t *testing.T, got assistant.ActionResultRenderer, found bool) {
				assert.True(t, found)
				assert.Same(t, renderer, got)
			},
		},
		"returns-not-found-when-missing-in-all-registries": {
			actionName:    "missing_action",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*assistant.MockActionRegistry) {
				registries[0].EXPECT().GetRenderer("missing_action").Return(nil, false).Once()
				registries[1].EXPECT().GetRenderer("missing_action").Return(nil, false).Once()
			},
			assertResult: func(t *testing.T, got assistant.ActionResultRenderer, found bool) {
				assert.False(t, found)
				assert.Nil(t, got)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registryMocks := newMockRegistries(t, tt.registriesLen)
			if tt.setMocks != nil {
				tt.setMocks(t, registryMocks)
			}

			registries := make([]assistant.ActionRegistry, 0, tt.registriesLen)
			for _, mock := range registryMocks {
				registries = append(registries, mock)
			}

			registry := NewCompositeActionRegistry(t.Context(), registries...)
			got, found := registry.GetRenderer(tt.actionName)
			tt.assertResult(t, got, found)
		})
	}
}

func newMockRegistries(t *testing.T, count int) []*assistant.MockActionRegistry {
	t.Helper()

	mocks := make([]*assistant.MockActionRegistry, 0, count)
	for range count {
		r := assistant.NewMockActionRegistry(t)
		mocks = append(mocks, r)
	}

	return mocks
}

type compositeMockRenderer struct{}

func (compositeMockRenderer) Render(_ assistant.ActionCall, _ assistant.Message) (assistant.Message, bool) {
	return assistant.Message{}, false
}
