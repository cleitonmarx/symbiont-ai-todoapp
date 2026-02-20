package assistant

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/assistant/actions"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAssistantActionManager(t *testing.T) {
	tests := map[string]struct {
		setupActions   func() []assistantActionVector
		embeddingModel string
		testFunc       func(t *testing.T, manager AssistantActionManager)
	}{
		"list-returns-all-actions": {
			setupActions: func() []assistantActionVector {
				action1 := domain.NewMockAssistantAction(t)
				action1.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos"))

				action2 := domain.NewMockAssistantAction(t)
				action2.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("create_todo"))

				return []assistantActionVector{
					{Action: action1},
					{Action: action2},
				}
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
		"status-message-returns-action-specific-message": {
			setupActions: func() []assistantActionVector {
				action := domain.NewMockAssistantAction(t)
				action.EXPECT().
					Definition().
					Return(mockAssistantActionDefinition("fetch_todos"))
				action.EXPECT().StatusMessage().Return("🔎 Fetching todos...")
				return []assistantActionVector{{Action: action}}
			},
			testFunc: func(t *testing.T, manager AssistantActionManager) {
				msg := manager.StatusMessage("fetch_todos")
				assert.Equal(t, "🔎 Fetching todos...", msg)
			},
		},
		"status-message-returns-default-when-action-not-found": {
			setupActions: func() []assistantActionVector { return []assistantActionVector{} },
			testFunc: func(t *testing.T, manager AssistantActionManager) {
				msg := manager.StatusMessage("unknown_action")
				assert.Equal(t, "⏳ Processing request...", msg)
			},
		},
		"execute-calls-correct-action": {
			setupActions: func() []assistantActionVector {
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
				return []assistantActionVector{{Action: action}}
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
		"execute-returns-error-for-unknown-action": {
			setupActions: func() []assistantActionVector { return []assistantActionVector{} },
			testFunc: func(t *testing.T, manager AssistantActionManager) {
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
			manager := NewAssistantActionManager(nil, "", tt.setupActions()...)
			tt.testFunc(t, manager)
		})
	}
}

func TestAssistantActionManager_List_And_StatusMessages(t *testing.T) {
	actionDetails := []assistantActionVector{
		{Action: actions.NewUIFiltersSetterAction()},
		{Action: actions.NewTodoFetcherAction(nil, nil, "")},
		{Action: actions.NewTodoCreatorAction(nil, nil, nil)},
		{Action: actions.NewTodoUpdaterAction(nil, nil)},
		{Action: actions.NewTodoDueDateUpdaterAction(nil, nil, nil)},
		{Action: actions.NewTodoDeleterAction(nil, nil)},
	}

	manager := NewAssistantActionManager(nil, "", actionDetails...)

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

func TestAssistantActionManager_List_IsSortedByName(t *testing.T) {
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
			actionVectors := make([]assistantActionVector, 0, len(tt.actionNames))
			for _, actionName := range tt.actionNames {
				actionVectors = append(actionVectors, assistantActionVector{
					Action: newStaticAssistantAction(actionName),
				})
			}

			manager := NewAssistantActionManager(nil, "", actionVectors...)
			actions := manager.List()
			require.Len(t, actions, len(tt.expectedNames))

			names := make([]string, 0, len(actions))
			for _, action := range actions {
				names = append(names, action.Name)
			}

			assert.Equal(t, tt.expectedNames, names)
		})
	}
}

func TestInitAssistantActionRegistry_Initialize(t *testing.T) {
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
				r, err := depend.Resolve[domain.AssistantActionRegistry]()
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

			i := InitAssistantActionRegistry{
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

func TestAssistantActionManager_ListRelevant(t *testing.T) {
	t.Run("returns-top-k-actions-by-similarity", func(t *testing.T) {
		mockEncoder := domain.NewMockSemanticEncoder(t)
		mockEncoder.EXPECT().
			VectorizeQuery(mock.Anything, "test-model", "mark as done and update title").
			Return(domain.EmbeddingVector{Vector: []float64{1, 0}}, nil).
			Once()

		manager := NewAssistantActionManager(mockEncoder, "test-model",
			assistantActionVector{
				Action:  newStaticAssistantAction("update_todo"),
				Vectors: []float64{1, 0},
			},
			assistantActionVector{
				Action:  newStaticAssistantAction("update_todo_due_date"),
				Vectors: []float64{0.95, 0.05},
			},
			assistantActionVector{
				Action:  newStaticAssistantAction("create_todo"),
				Vectors: []float64{0.8, 0.2},
			},
			assistantActionVector{
				Action:  newStaticAssistantAction("fetch_todos"),
				Vectors: []float64{0.6, 0.4},
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

		manager := NewAssistantActionManager(mockEncoder, "test-model",
			assistantActionVector{
				Action:  newStaticAssistantAction("fetch_todos"),
				Vectors: []float64{1, 0},
			},
			assistantActionVector{
				Action:  newStaticAssistantAction("set_ui_filters"),
				Vectors: []float64{0.9, 0.1},
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

		manager := NewAssistantActionManager(mockEncoder, "test-model",
			assistantActionVector{
				Action:  newStaticAssistantAction("create_todo"),
				Vectors: []float64{1, 0},
			},
			assistantActionVector{
				Action:  newStaticAssistantAction("update_todo"),
				Vectors: []float64{1, 0},
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
