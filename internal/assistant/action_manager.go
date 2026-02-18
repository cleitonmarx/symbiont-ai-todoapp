package assistant

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/assistant/actions"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/cleitonmarx/symbiont/depend"
)

// AssistantActionManager manages assistant actions.
type AssistantActionManager struct {
	actions map[string]domain.AssistantAction
}

// NewAssistantActionManager creates an assistant action registry.
func NewAssistantActionManager(actions ...domain.AssistantAction) AssistantActionManager {
	actionMap := make(map[string]domain.AssistantAction)
	for _, action := range actions {
		actionMap[action.Definition().Name] = action
	}
	return AssistantActionManager{
		actions: actionMap,
	}
}

// Execute invokes the appropriate action.
func (m AssistantActionManager) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	action, exists := m.actions[call.Name]
	if !exists {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"unknown_tool","details":"Tool '%s' is not registered."}`, call.Name),
		}
	}

	return action.Execute(ctx, call, conversationHistory)
}

// StatusMessage returns a status message about the action execution.
func (m AssistantActionManager) StatusMessage(actionName string) string {
	if action, ok := m.actions[actionName]; ok {
		if msg := action.StatusMessage(); msg != "" {
			return msg
		}
	}
	return "⏳ Processing request..."
}

// List returns all available assistant action definitions.
func (m AssistantActionManager) List() []domain.AssistantActionDefinition {
	res := make([]domain.AssistantActionDefinition, 0, len(m.actions))
	for _, action := range m.actions {
		res = append(res, action.Definition())
	}
	return res
}

type InitAssistantActionRegistry struct {
	Uow             domain.UnitOfWork          `resolve:""`
	TodoCreator     usecases.TodoCreator       `resolve:""`
	TodoUpdater     usecases.TodoUpdater       `resolve:""`
	TodoDeleter     usecases.TodoDeleter       `resolve:""`
	TodoRepo        domain.TodoRepository      `resolve:""`
	SemanticEncoder domain.SemanticEncoder     `resolve:""`
	TimeProvider    domain.CurrentTimeProvider `resolve:""`
	EmbeddingModel  string                     `config:"LLM_EMBEDDING_MODEL"`
}

func (i InitAssistantActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	actions := []domain.AssistantAction{
		actions.NewUIFiltersSetterAction(),
		actions.NewTodoFetcherAction(
			i.TodoRepo,
			i.SemanticEncoder,
			i.TimeProvider,
			i.EmbeddingModel,
		),
		actions.NewTodoCreatorAction(
			i.Uow,
			i.TodoCreator,
			i.TimeProvider,
		),
		actions.NewTodoUpdaterAction(
			i.Uow,
			i.TodoUpdater,
		),
		actions.NewTodoDueDateUpdaterAction(
			i.Uow,
			i.TodoUpdater,
			i.TimeProvider,
		),
		actions.NewTodoDeleterAction(
			i.Uow,
			i.TodoDeleter,
		),
	}

	actionRegistry := NewAssistantActionManager(actions...)

	depend.Register[domain.AssistantActionRegistry](actionRegistry)
	return ctx, nil
}
