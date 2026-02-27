package composite

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCompositeActionRegistry_Execute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		call          domain.AssistantActionCall
		history       []domain.AssistantMessage
		registriesLen int
		setMocks      func(t *testing.T, registries []*domain.MockAssistantActionRegistry)
		assertMessage func(t *testing.T, message domain.AssistantMessage)
	}{
		"routes-execution-to-matching-action": {
			call:          domain.AssistantActionCall{ID: "call-1", Name: "fetch_todos", Input: "{}"},
			history:       []domain.AssistantMessage{{Role: domain.ChatRole_User, Content: "hello"}},
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*domain.MockAssistantActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("fetch_todos").
					Return(domain.AssistantActionDefinition{Name: "fetch_todos"}, true).
					Once()
				registries[0].EXPECT().
					Execute(
						mock.Anything,
						domain.AssistantActionCall{ID: "call-1", Name: "fetch_todos", Input: "{}"},
						[]domain.AssistantMessage{{Role: domain.ChatRole_User, Content: "hello"}},
					).
					Return(domain.AssistantMessage{
						Role:         domain.ChatRole_Tool,
						Content:      "todos",
						ActionCallID: common.Ptr("call-1"),
					}).
					Once()
			},
			assertMessage: func(t *testing.T, message domain.AssistantMessage) {
				require.NotNil(t, message.ActionCallID)
				assert.Equal(t, domain.ChatRole_Tool, message.Role)
				assert.Equal(t, "todos", message.Content)
				assert.Equal(t, "call-1", *message.ActionCallID)
			},
		},
		"returns-error-when-action-is-not-registered": {
			call:          domain.AssistantActionCall{ID: "call-2", Name: "unknown_action", Input: "{}"},
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*domain.MockAssistantActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("unknown_action").
					Return(domain.AssistantActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("unknown_action").
					Return(domain.AssistantActionDefinition{}, false).
					Once()
			},
			assertMessage: func(t *testing.T, message domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, message.Role)
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
			registries := make([]domain.AssistantActionRegistry, 0, tt.registriesLen)
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
		setMocks      func(t *testing.T, registries []*domain.MockAssistantActionRegistry)
		assertResult  func(t *testing.T, definition domain.AssistantActionDefinition, found bool)
	}{
		"returns-definition-from-first-registry": {
			actionName:    "fetch_todos",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*domain.MockAssistantActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("fetch_todos").
					Return(domain.AssistantActionDefinition{Name: "fetch_todos", Description: "from first"}, true).
					Once()
			},
			assertResult: func(t *testing.T, definition domain.AssistantActionDefinition, found bool) {
				assert.True(t, found)
				assert.Equal(t, "fetch_todos", definition.Name)
				assert.Equal(t, "from first", definition.Description)
			},
		},
		"falls-back-to-next-registry-when-first-misses": {
			actionName:    "update_todos",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*domain.MockAssistantActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("update_todos").
					Return(domain.AssistantActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("update_todos").
					Return(domain.AssistantActionDefinition{Name: "update_todos", Description: "from second"}, true).
					Once()
			},
			assertResult: func(t *testing.T, definition domain.AssistantActionDefinition, found bool) {
				assert.True(t, found)
				assert.Equal(t, "update_todos", definition.Name)
				assert.Equal(t, "from second", definition.Description)
			},
		},
		"returns-not-found-when-missing-in-all-registries": {
			actionName:    "missing_action",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*domain.MockAssistantActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("missing_action").
					Return(domain.AssistantActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("missing_action").
					Return(domain.AssistantActionDefinition{}, false).
					Once()
			},
			assertResult: func(t *testing.T, definition domain.AssistantActionDefinition, found bool) {
				assert.False(t, found)
				assert.Equal(t, domain.AssistantActionDefinition{}, definition)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registryMocks := newMockRegistries(t, tt.registriesLen)
			if tt.setMocks != nil {
				tt.setMocks(t, registryMocks)
			}

			registries := make([]domain.AssistantActionRegistry, 0, tt.registriesLen)
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
		setMocks      func(t *testing.T, registries []*domain.MockAssistantActionRegistry)
		expected      string
	}{
		"returns-action-status-message-when-action-exists": {
			actionName:    "search_web",
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*domain.MockAssistantActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("search_web").
					Return(domain.AssistantActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("search_web").
					Return(domain.AssistantActionDefinition{Name: "search_web"}, true).
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
			setMocks: func(_ *testing.T, registries []*domain.MockAssistantActionRegistry) {
				registries[0].EXPECT().
					GetDefinition("missing_action").
					Return(domain.AssistantActionDefinition{}, false).
					Once()
				registries[1].EXPECT().
					GetDefinition("missing_action").
					Return(domain.AssistantActionDefinition{}, false).
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

			registries := make([]domain.AssistantActionRegistry, 0, tt.registriesLen)
			for _, mock := range registryMocks {
				registries = append(registries, mock)
			}

			registry := NewCompositeActionRegistry(t.Context(), registries...)
			assert.Equal(t, tt.expected, registry.StatusMessage(tt.actionName))
		})
	}
}

func newMockRegistries(t *testing.T, count int) []*domain.MockAssistantActionRegistry {
	t.Helper()

	mocks := make([]*domain.MockAssistantActionRegistry, 0, count)
	for range count {
		r := domain.NewMockAssistantActionRegistry(t)
		mocks = append(mocks, r)
	}

	return mocks
}

func TestInitCompositeActionRegistry_Initialize(t *testing.T) {
	t.Parallel()

	r := &InitCompositeActionRegistry{}
	ctx, err := r.Initialize(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, ctx)
	dep, err := depend.Resolve[domain.AssistantActionRegistry]()
	require.NoError(t, err)
	assert.IsType(t, CompositeActionRegistry{}, dep)
}
