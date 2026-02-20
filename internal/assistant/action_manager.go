package assistant

import (
	"context"
	"fmt"
	"sort"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/assistant/actions"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/cleitonmarx/symbiont/depend"
)

const (
	defaultRelevantActionsTopK     = 3
	defaultRelevantActionsMinScore = 0.35
)

// assistantActionVector holds an assistant action and its corresponding
// vector embedding for relevance scoring.
type assistantActionVector struct {
	Action  domain.AssistantAction
	Vectors []float64
}

// AssistantActionManager manages assistant actions.
type AssistantActionManager struct {
	se             domain.SemanticEncoder
	embeddingModel string
	actionsDetails map[string]assistantActionVector
}

// NewAssistantActionManager creates an assistant action registry.
func NewAssistantActionManager(se domain.SemanticEncoder, embeddingModel string, actionsDetails ...assistantActionVector) AssistantActionManager {
	actionMap := make(map[string]assistantActionVector)
	for _, action := range actionsDetails {
		actionMap[action.Action.Definition().Name] = action
	}

	return AssistantActionManager{
		se:             se,
		embeddingModel: embeddingModel,
		actionsDetails: actionMap,
	}
}

// Execute invokes the appropriate action.
func (m AssistantActionManager) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	details, exists := m.actionsDetails[call.Name]
	if !exists {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"unknown_action","details":"Action '%s' is not registered."}`, call.Name),
		}
	}

	return details.Action.Execute(ctx, call, conversationHistory)
}

// StatusMessage returns a status message about the action execution.
func (m AssistantActionManager) StatusMessage(actionName string) string {
	if action, ok := m.actionsDetails[actionName]; ok {
		if msg := action.Action.StatusMessage(); msg != "" {
			return msg
		}
	}
	return "⏳ Processing request..."
}

// List returns all available assistant action definitions.
func (m AssistantActionManager) List() []domain.AssistantActionDefinition {
	res := make([]domain.AssistantActionDefinition, 0, len(m.actionsDetails))
	for _, action := range m.actionsDetails {
		res = append(res, action.Action.Definition())
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res
}

func (m AssistantActionManager) ListRelevant(ctx context.Context, userInput string) []domain.AssistantActionDefinition {
	allActions := m.List()

	queryVector, err := m.se.VectorizeQuery(ctx, m.embeddingModel, userInput)
	if err != nil || len(queryVector.Vector) == 0 {
		return allActions
	}

	type scoredAction struct {
		definition domain.AssistantActionDefinition
		score      float64
	}

	scoredActions := make([]scoredAction, 0, len(m.actionsDetails))
	for _, actionDetail := range m.actionsDetails {
		score, ok := common.CosineSimilarity(queryVector.Vector, actionDetail.Vectors)
		if !ok || score < defaultRelevantActionsMinScore {
			continue
		}

		scoredActions = append(scoredActions, scoredAction{
			definition: actionDetail.Action.Definition(),
			score:      score,
		})
	}

	if len(scoredActions) == 0 {
		return allActions
	}

	sort.Slice(scoredActions, func(i, j int) bool {
		if scoredActions[i].score == scoredActions[j].score {
			return scoredActions[i].definition.Name < scoredActions[j].definition.Name
		}
		return scoredActions[i].score > scoredActions[j].score
	})

	limit := min(len(scoredActions), defaultRelevantActionsTopK)

	relevant := make([]domain.AssistantActionDefinition, 0, limit)
	for i := range limit {
		relevant = append(relevant, scoredActions[i].definition)
	}
	return relevant
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

	actionVectors, err := generateActionVectors(ctx, actions, i.SemanticEncoder, i.EmbeddingModel)
	if err != nil {
		return ctx, fmt.Errorf("failed to build assistant action vectors: %w", err)
	}

	actionRegistry := NewAssistantActionManager(i.SemanticEncoder, i.EmbeddingModel, actionVectors...)
	depend.Register[domain.AssistantActionRegistry](actionRegistry)
	return ctx, nil
}

// generateActionVectors generates vector embeddings for a list of assistant actions.
func generateActionVectors(ctx context.Context, actions []domain.AssistantAction, encoder domain.SemanticEncoder, embeddingModel string) ([]assistantActionVector, error) {
	var details []assistantActionVector
	for _, action := range actions {
		vector, err := encoder.VectorizeAssistantActionDefinition(ctx, embeddingModel, action.Definition())
		if err != nil {
			return nil, fmt.Errorf("failed to vectorize action '%s': %w", action.Definition().Name, err)
		}
		details = append(details, assistantActionVector{
			Action:  action,
			Vectors: vector.Vector,
		})
	}
	return details, nil
}
