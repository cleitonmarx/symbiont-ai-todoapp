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
	tools map[string]domain.AssistantAction
}

// NewAssistantActionManager creates an assistant action registry.
func NewAssistantActionManager(tools ...domain.AssistantAction) AssistantActionManager {
	toolMap := make(map[string]domain.AssistantAction)
	for _, tool := range tools {
		toolMap[tool.Definition().Name] = tool
	}
	return AssistantActionManager{
		tools: toolMap,
	}
}

// Execute invokes the appropriate action.
func (m AssistantActionManager) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	tool, exists := m.tools[call.Name]
	if !exists {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"unknown_tool","details":"Tool '%s' is not registered."}`, call.Name),
		}
	}

	return tool.Execute(ctx, call, conversationHistory)
}

// StatusMessage returns a status message about the action execution.
func (m AssistantActionManager) StatusMessage(actionName string) string {
	if tool, ok := m.tools[actionName]; ok {
		if msg := tool.StatusMessage(); msg != "" {
			return msg
		}
	}
	return "⏳ Processing request..."
}

// List returns all available assistant action definitions.
func (m AssistantActionManager) List() []domain.AssistantActionDefinition {
	res := make([]domain.AssistantActionDefinition, 0, len(m.tools))
	for _, tool := range m.tools {
		res = append(res, tool.Definition())
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
	tools := []domain.AssistantAction{
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

	actionRegistry := NewAssistantActionManager(tools...)

	depend.Register[domain.AssistantActionRegistry](actionRegistry)
	return ctx, nil
}
