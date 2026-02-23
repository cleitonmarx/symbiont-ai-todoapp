package composite

import (
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCompositeActionRegistry_Execute(t *testing.T) {
	tests := map[string]struct {
		call          domain.AssistantActionCall
		history       []domain.AssistantMessage
		registriesLen int
		setMocks      func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry)
		assertMessage func(t *testing.T, message domain.AssistantMessage)
	}{
		"routes-execution-to-matching-action": {
			call:          domain.AssistantActionCall{ID: "call-1", Name: "fetch_todos", Input: "{}"},
			history:       []domain.AssistantMessage{{Role: domain.ChatRole_User, Content: "hello"}},
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				otherAction := newMockAction(t, "other_action")
				action := newMockAction(t, "fetch_todos")
				action.EXPECT().
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

				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(otherAction, nil)}).
					Once()
				registries[1].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(action, nil)}).
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
			setMocks: func(_ *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{}).
					Once()
				registries[1].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{}).
					Once()
			},
			assertMessage: func(t *testing.T, message domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, message.Role)
				assert.Equal(t, "error: no registry found for action 'unknown_action'", message.Content)
			},
		},
		"last-registry-wins-for-duplicate-action-names": {
			call:          domain.AssistantActionCall{ID: "call-3", Name: "duplicate", Input: "{}"},
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				firstAction := newMockAction(t, "duplicate")
				secondAction := newMockAction(t, "duplicate")
				secondAction.EXPECT().
					Execute(mock.Anything, mock.Anything, mock.Anything).
					Return(domain.AssistantMessage{Role: domain.ChatRole_Tool, Content: "second"}).
					Once()

				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(firstAction, nil)}).
					Once()
				registries[1].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(secondAction, nil)}).
					Once()
			},
			assertMessage: func(t *testing.T, message domain.AssistantMessage) {
				assert.Equal(t, "second", message.Content)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registryMocks, registries := newMockRegistries(t, tt.registriesLen)
			if tt.setMocks != nil {
				tt.setMocks(t, registryMocks)
			}

			registry := NewCompositeActionRegistry(t.Context(), nil, "", registries...)
			message := registry.Execute(t.Context(), tt.call, tt.history)
			if tt.assertMessage != nil {
				tt.assertMessage(t, message)
			}
		})
	}
}

func TestCompositeActionRegistry_StatusMessage(t *testing.T) {
	tests := map[string]struct {
		actionName    string
		registriesLen int
		setMocks      func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry)
		expected      string
	}{
		"returns-action-status-message-when-action-exists": {
			actionName:    "search_web",
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				otherAction := newMockAction(t, "other_action")
				action := newMockAction(t, "search_web")
				action.EXPECT().StatusMessage().Return("🔎 searching...").Once()

				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(otherAction, nil)}).
					Once()
				registries[1].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(action, nil)}).
					Once()
			},
			expected: "🔎 searching...",
		},
		"returns-default-status-message-when-action-does-not-exist": {
			actionName:    "missing_action",
			registriesLen: 2,
			setMocks: func(_ *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{}).
					Once()
				registries[1].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{}).
					Once()
			},
			expected: "⏳ Processing request...",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registryMocks, registries := newMockRegistries(t, tt.registriesLen)
			if tt.setMocks != nil {
				tt.setMocks(t, registryMocks)
			}

			registry := NewCompositeActionRegistry(t.Context(), nil, "", registries...)
			assert.Equal(t, tt.expected, registry.StatusMessage(tt.actionName))
		})
	}
}

func TestCompositeActionRegistry_ListRelevant(t *testing.T) {
	tests := map[string]struct {
		queryVector     domain.EmbeddingVector
		queryErr        error
		registriesLen   int
		setMocks        func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry)
		expectedNames   []string
		expectedOrdered bool
	}{
		"returns-all-actions-when-query-vectorization-fails": {
			queryErr:      errors.New("vectorization failed"),
			queryVector:   domain.EmbeddingVector{},
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				createAction := newMockAction(t, "create_todo")
				deleteAction := newMockAction(t, "delete_todo")

				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(createAction, []float64{1, 0})}).
					Once()
				registries[1].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(deleteAction, []float64{0, 1})}).
					Once()
			},
			expectedNames: []string{"create_todo", "delete_todo"},
		},
		"returns-all-actions-when-query-vector-is-empty": {
			queryVector:   domain.EmbeddingVector{Vector: nil},
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				updateAction := newMockAction(t, "update_todo")
				fetchAction := newMockAction(t, "fetch_todos")

				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(updateAction, []float64{1, 0})}).
					Once()
				registries[1].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{actionEmbedding(fetchAction, []float64{0, 1})}).
					Once()
			},
			expectedNames: []string{"update_todo", "fetch_todos"},
		},
		"filters-by-score-and-limits-to-top-k": {
			queryVector:   domain.EmbeddingVector{Vector: []float64{1, 0}},
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				alpha := newMockAction(t, "alpha")
				beta := newMockAction(t, "beta")
				gamma := newMockAction(t, "gamma")
				epsilon := newMockAction(t, "epsilon")
				delta := newMockAction(t, "delta")

				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{
						actionEmbedding(alpha, []float64{1, 0}),
						actionEmbedding(beta, []float64{0.8, 0.6}),
					}).
					Once()
				registries[1].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{
						actionEmbedding(gamma, []float64{0.6, 0.8}),
						actionEmbedding(epsilon, []float64{0.4, 0.916515138991168}),
						actionEmbedding(delta, []float64{0.2, 0.9797958971132712}),
					}).
					Once()
			},
			expectedNames:   []string{"alpha", "beta", "gamma"},
			expectedOrdered: true,
		},
		"returns-all-actions-when-no-scores-pass-threshold": {
			queryVector:   domain.EmbeddingVector{Vector: []float64{1, 0}},
			registriesLen: 2,
			setMocks: func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				lowA := newMockAction(t, "low_a")
				lowB := newMockAction(t, "low_b")
				lowC := newMockAction(t, "low_c")

				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{
						actionEmbedding(lowA, []float64{0.2, 0.9797958971132712}),
					}).
					Once()
				registries[1].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{
						actionEmbedding(lowB, []float64{0.1, 0.99498743710662}),
						actionEmbedding(lowC, []float64{0, 1}),
					}).
					Once()
			},
			expectedNames: []string{"low_a", "low_b", "low_c"},
		},
		"uses-action-name-as-tie-breaker-for-equal-scores": {
			queryVector:   domain.EmbeddingVector{Vector: []float64{1, 1}},
			registriesLen: 1,
			setMocks: func(t *testing.T, registries []*actionregistry.MockEmbeddingActionRegistry) {
				bravo := newMockAction(t, "bravo")
				alpha := newMockAction(t, "alpha")
				charlie := newMockAction(t, "charlie")

				registries[0].EXPECT().
					ListEmbeddings(mock.Anything).
					Return([]actionregistry.ActionEmbedding{
						actionEmbedding(bravo, []float64{1, 0}),
						actionEmbedding(alpha, []float64{0, 1}),
						actionEmbedding(charlie, []float64{1, 1}),
					}).
					Once()
			},
			expectedNames:   []string{"charlie", "alpha", "bravo"},
			expectedOrdered: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registryMocks, registries := newMockRegistries(t, tt.registriesLen)
			if tt.setMocks != nil {
				tt.setMocks(t, registryMocks)
			}

			semanticEncoder := domain.NewMockSemanticEncoder(t)
			semanticEncoder.EXPECT().
				VectorizeQuery(mock.Anything, "embedding-model", "user request").
				Return(tt.queryVector, tt.queryErr).
				Once()

			registry := NewCompositeActionRegistry(t.Context(), semanticEncoder, "embedding-model", registries...)
			definitions := registry.ListRelevant(t.Context(), "user request")
			gotNames := definitionsNames(definitions)

			if tt.expectedOrdered {
				assert.Equal(t, tt.expectedNames, gotNames)
				return
			}

			assert.ElementsMatch(t, tt.expectedNames, gotNames)
		})
	}
}

func newMockAction(t *testing.T, name string) *domain.MockAssistantAction {
	t.Helper()

	action := domain.NewMockAssistantAction(t)
	action.EXPECT().Definition().Return(domain.AssistantActionDefinition{
		Name:        name,
		Description: "test action " + name,
		Input: domain.AssistantActionInput{
			Type: "object",
		},
	}).Maybe()

	return action
}

func newMockRegistries(t *testing.T, count int) ([]*actionregistry.MockEmbeddingActionRegistry, []actionregistry.EmbeddingActionRegistry) {
	t.Helper()

	mocks := make([]*actionregistry.MockEmbeddingActionRegistry, 0, count)
	registries := make([]actionregistry.EmbeddingActionRegistry, 0, count)
	for range count {
		r := actionregistry.NewMockEmbeddingActionRegistry(t)
		mocks = append(mocks, r)
		registries = append(registries, r)
	}

	return mocks, registries
}

func actionEmbedding(action domain.AssistantAction, embedding []float64) actionregistry.ActionEmbedding {
	return actionregistry.ActionEmbedding{
		Action:    action,
		Embedding: embedding,
	}
}

func definitionsNames(definitions []domain.AssistantActionDefinition) []string {
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.Name)
	}
	return names
}

func TestInitCompositeActionRegistry_Initialize(t *testing.T) {
	localMock := actionregistry.NewMockEmbeddingActionRegistry(t)
	localMock.EXPECT().ListEmbeddings(mock.Anything).Return([]actionregistry.ActionEmbedding{}).Once()
	mcpMock := actionregistry.NewMockEmbeddingActionRegistry(t)
	mcpMock.EXPECT().ListEmbeddings(mock.Anything).Return([]actionregistry.ActionEmbedding{}).Once()

	r := &InitCompositeActionRegistry{
		Local: localMock,
		MCP:   mcpMock,
	}
	ctx, err := r.Initialize(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, ctx)
	dep, err := depend.Resolve[domain.AssistantActionRegistry]()
	require.NoError(t, err)
	assert.IsType(t, CompositeActionRegistry{}, dep)
}
