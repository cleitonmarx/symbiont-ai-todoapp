package local

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry/local/actions"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestActionRegistry(t *testing.T) {
	tests := map[string]struct {
		setupActions   func() []actionregistry.ActionEmbedding
		embeddingModel string
		testFunc       func(t *testing.T, manager LocalRegistry)
	}{
		"list-returns-all-actions": {
			setupActions: func() []actionregistry.ActionEmbedding {
				action1 := domain.NewMockAssistantAction(t)
				action1.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos"))

				action2 := domain.NewMockAssistantAction(t)
				action2.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("create_todo"))

				return []actionregistry.ActionEmbedding{
					{Action: action1},
					{Action: action2},
				}
			},
			testFunc: func(t *testing.T, manager LocalRegistry) {
				actions := manager.ListEmbeddings(t.Context())
				assert.Len(t, actions, 2)
				names := []string{}
				for _, action := range actions {
					names = append(names, action.Action.Definition().Name)
				}
				assert.ElementsMatch(t, []string{"fetch_todos", "create_todo"}, names)
			},
		},
		"status-message-returns-action-specific-message": {
			setupActions: func() []actionregistry.ActionEmbedding {
				action := domain.NewMockAssistantAction(t)
				action.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos"))
				action.EXPECT().StatusMessage().Return("🔎 Fetching todos...")
				return []actionregistry.ActionEmbedding{{Action: action}}
			},
			testFunc: func(t *testing.T, manager LocalRegistry) {
				msg := manager.StatusMessage("fetch_todos")
				assert.Equal(t, "🔎 Fetching todos...", msg)
			},
		},
		"status-message-returns-default-when-action-not-found": {
			setupActions: func() []actionregistry.ActionEmbedding { return []actionregistry.ActionEmbedding{} },
			testFunc: func(t *testing.T, manager LocalRegistry) {
				msg := manager.StatusMessage("unknown_action")
				assert.Equal(t, "⏳ Processing request...", msg)
			},
		},
		"execute-calls-correct-action": {
			setupActions: func() []actionregistry.ActionEmbedding {
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
				return []actionregistry.ActionEmbedding{{Action: action}}
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
			setupActions: func() []actionregistry.ActionEmbedding { return []actionregistry.ActionEmbedding{} },
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

func TestActionRegistry_List_And_StatusMessages(t *testing.T) {
	vectorizedActions := []actionregistry.ActionEmbedding{
		{Action: actions.NewUIFiltersSetterAction()},
		{Action: actions.NewTodoFetcherAction(nil, nil, "")},
		{Action: actions.NewBulkTodoCreatorAction(nil, nil, nil)},
		{Action: actions.NewBulkTodoUpdaterAction(nil, nil)},
		{Action: actions.NewBulkTodoDueDateUpdaterAction(nil, nil, nil)},
		{Action: actions.NewBulkTodoDeleterAction(nil, nil)},
	}

	manager := NewActionRegistry(nil, "", vectorizedActions...)

	actions := manager.ListEmbeddings(t.Context())
	require.Len(t, actions, 10)

	names := make([]string, 0, len(vectorizedActions))
	for _, vectorizedAction := range vectorizedActions {
		names = append(names, vectorizedAction.Action.Definition().Name)
	}

	assert.ElementsMatch(t, []string{
		"set_ui_filters",
		"fetch_todos",
		"create_todos",
		"update_todos",
		"update_todos_due_date",
		"delete_todos",
	}, names)

	statusMessages := []string{}
	for _, name := range names {
		statusMessages = append(statusMessages, manager.StatusMessage(name))
	}

	assert.ElementsMatch(t, []string{
		"🎛️ Applying filters...",
		"🔎 Fetching todos...",
		"📝 Creating your todos...",
		"✏️ Updating your todos...",
		"📅 Updating the due date...",
		"🗑️ Deleting todos...",
	}, statusMessages)
}

func TestActionRegistry_List_IsSortedByName(t *testing.T) {
	tests := map[string]struct {
		actionNames   []string
		expectedNames []string
	}{
		"sorts-actions-by-name": {
			actionNames:   []string{"update_todo", "create_todo", "delete_todo"},
			expectedNames: []string{"create_todo", "delete_todo", "update_todo"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			actionVectors := make([]actionregistry.ActionEmbedding, 0, len(tt.actionNames))
			for _, actionName := range tt.actionNames {
				actionVectors = append(actionVectors, actionregistry.ActionEmbedding{
					Action: newStaticAssistantAction(actionName),
				})
			}

			manager := NewActionRegistry(nil, "", actionVectors...)
			vectorizedActions := manager.ListEmbeddings(t.Context())
			require.Len(t, vectorizedActions, len(tt.expectedNames))

			names := make([]string, 0, len(vectorizedActions))
			for _, action := range vectorizedActions {
				names = append(names, action.Action.Definition().Name)
			}

			assert.Equal(t, tt.expectedNames, names)
		})
	}
}

func TestInitActionRegistry_Initialize(t *testing.T) {
	tests := map[string]struct {
		setupMock    func(*domain.MockSemanticEncoder)
		expectError  bool
		validateFunc func(*testing.T, context.Context)
	}{
		"successfully-initializes-registry": {
			setupMock: func(mockEncoder *domain.MockSemanticEncoder) {
				mockEncoder.EXPECT().
					VectorizeAssistantActionDefinition(
						mock.Anything,
						"test-model",
						mock.AnythingOfType("domain.AssistantActionDefinition"),
					).
					Return(domain.EmbeddingVector{
						Vector:      []float64{0.1, 0.2, 0.3},
						TotalTokens: 3,
					}, nil).Times(6)
			},
			expectError: false,
			validateFunc: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
				r, err := depend.ResolveNamed[actionregistry.EmbeddingActionRegistry]("local")
				require.NoError(t, err)
				assert.NotNil(t, r)
			},
		},
		"embedding-error-during-initialization": {
			setupMock: func(mockEncoder *domain.MockSemanticEncoder) {
				mockEncoder.EXPECT().
					VectorizeAssistantActionDefinition(
						mock.Anything,
						"test-model",
						mock.AnythingOfType("domain.AssistantActionDefinition"),
					).
					Return(domain.EmbeddingVector{}, assert.AnError).Times(1)
			},
			expectError: true,
			validateFunc: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockEncoder := domain.NewMockSemanticEncoder(t)
			tt.setupMock(mockEncoder)

			i := InitLocalActionRegistry{
				SemanticEncoder: mockEncoder,
				EmbeddingModel:  "test-model",
			}

			ctx, err := i.Initialize(t.Context())

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, ctx)
			}
		})
	}
}

func TestActionRegistry_ListRelevant(t *testing.T) {
	t.Run("returns-top-k-actions-by-similarity", func(t *testing.T) {
		mockEncoder := domain.NewMockSemanticEncoder(t)
		mockEncoder.EXPECT().
			VectorizeQuery(mock.Anything, "test-model", "mark as done and update title").
			Return(domain.EmbeddingVector{Vector: []float64{1, 0}}, nil).
			Once()

		manager := NewActionRegistry(mockEncoder, "test-model",
			actionregistry.ActionEmbedding{
				Action:    newStaticAssistantAction("update_todo"),
				Embedding: []float64{1, 0},
			},
			actionregistry.ActionEmbedding{
				Action:    newStaticAssistantAction("update_todo_due_date"),
				Embedding: []float64{0.95, 0.05},
			},
			actionregistry.ActionEmbedding{
				Action:    newStaticAssistantAction("create_todo"),
				Embedding: []float64{0.8, 0.2},
			},
			actionregistry.ActionEmbedding{
				Action:    newStaticAssistantAction("fetch_todos"),
				Embedding: []float64{0.6, 0.4},
			},
		)

		relevant := manager.ListRelevant(t.Context(), "mark as done and update title")
		require.Len(t, relevant, 3)
		assert.Equal(t, "update_todo", relevant[0].Name)
		assert.Equal(t, "update_todo_due_date", relevant[1].Name)
		assert.Equal(t, "create_todo", relevant[2].Name)
	})

	t.Run("falls-back-to-all-actions-when-vectorize-query-fails", func(t *testing.T) {
		mockEncoder := domain.NewMockSemanticEncoder(t)
		mockEncoder.EXPECT().
			VectorizeQuery(mock.Anything, "test-model", "show overdue items").
			Return(domain.EmbeddingVector{}, errors.New("embedding unavailable")).
			Once()

		manager := NewActionRegistry(mockEncoder, "test-model",
			actionregistry.ActionEmbedding{
				Action:    newStaticAssistantAction("fetch_todos"),
				Embedding: []float64{1, 0},
			},
			actionregistry.ActionEmbedding{
				Action:    newStaticAssistantAction("set_ui_filters"),
				Embedding: []float64{0.9, 0.1},
			},
		)

		relevant := manager.ListRelevant(t.Context(), "show overdue items")
		require.Len(t, relevant, 2)
		names := []string{relevant[0].Name, relevant[1].Name}
		assert.ElementsMatch(t, []string{"fetch_todos", "set_ui_filters"}, names)
	})

	t.Run("falls-back-to-all-actions-when-no-action-meets-threshold", func(t *testing.T) {
		mockEncoder := domain.NewMockSemanticEncoder(t)
		mockEncoder.EXPECT().
			VectorizeQuery(mock.Anything, "test-model", "delete everything").
			Return(domain.EmbeddingVector{Vector: []float64{0, 1}}, nil).
			Once()

		manager := NewActionRegistry(mockEncoder, "test-model",
			actionregistry.ActionEmbedding{
				Action:    newStaticAssistantAction("create_todo"),
				Embedding: []float64{1, 0},
			},
			actionregistry.ActionEmbedding{
				Action:    newStaticAssistantAction("update_todo"),
				Embedding: []float64{1, 0},
			},
		)

		relevant := manager.ListRelevant(t.Context(), "delete everything")
		require.Len(t, relevant, 2)
		names := []string{relevant[0].Name, relevant[1].Name}
		assert.ElementsMatch(t, []string{"create_todo", "update_todo"}, names)
	})

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

type staticAssistantAction struct {
	definition domain.AssistantActionDefinition
}

func newStaticAssistantAction(name string) staticAssistantAction {
	return staticAssistantAction{
		definition: mockAssistantActionDefinition(name),
	}
}

func (a staticAssistantAction) Definition() domain.AssistantActionDefinition { return a.definition }
func (a staticAssistantAction) StatusMessage() string                        { return "" }
func (a staticAssistantAction) Execute(context.Context, domain.AssistantActionCall, []domain.AssistantMessage) domain.AssistantMessage {
	return domain.AssistantMessage{}
}
