package local

import (
	"context"
	"fmt"
	"sort"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry/local/actions"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/cleitonmarx/symbiont/depend"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultRelevantActionsTopK     = 3
	defaultRelevantActionsMinScore = 0.35
)

// LocalRegistry manages a set of assistant actions defined within the todo application.
type LocalRegistry struct {
	se                    domain.SemanticEncoder
	embeddingModel        string
	embeddingByActionName map[string]actionregistry.ActionEmbedding
}

// NewActionRegistry creates a local assistant action registry.
func NewActionRegistry(se domain.SemanticEncoder, embeddingModel string, actionVectorList ...actionregistry.ActionEmbedding) LocalRegistry {
	actionEmbeddingMap := make(map[string]actionregistry.ActionEmbedding)
	for _, actionVector := range actionVectorList {
		actionEmbeddingMap[actionVector.Action.Definition().Name] = actionVector
	}

	return LocalRegistry{
		se:                    se,
		embeddingModel:        embeddingModel,
		embeddingByActionName: actionEmbeddingMap,
	}
}

// Execute invokes the appropriate action.
func (r LocalRegistry) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	spanCtx, span := telemetry.Start(ctx, trace.WithAttributes(
		attribute.String("assistant_action", call.Name),
	))
	defer span.End()
	details, exists := r.embeddingByActionName[call.Name]
	if !exists {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"unknown_action","details":"Action '%s' is not registered."}`, call.Name),
		}
	}

	return details.Action.Execute(spanCtx, call, conversationHistory)
}

// GetDefinition returns one action definition by name.
func (r LocalRegistry) GetDefinition(actionName string) (domain.AssistantActionDefinition, bool) {
	details, exists := r.embeddingByActionName[actionName]
	if !exists {
		return domain.AssistantActionDefinition{}, false
	}
	return details.Action.Definition(), true
}

// StatusMessage returns a status message about the action execution.
func (r LocalRegistry) StatusMessage(actionName string) string {
	if action, ok := r.embeddingByActionName[actionName]; ok {
		if msg := action.Action.StatusMessage(); msg != "" {
			return msg
		}
	}
	return "⏳ Processing request..."
}

// ListEmbeddings returns all available assistant action definitions along with their vector embeddings for relevance scoring.
func (r LocalRegistry) ListEmbeddings(ctx context.Context) []actionregistry.ActionEmbedding {
	_, span := telemetry.Start(ctx)
	defer span.End()

	res := make([]actionregistry.ActionEmbedding, 0, len(r.embeddingByActionName))
	for _, action := range r.embeddingByActionName {
		res = append(res, action)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Action.Definition().Name < res[j].Action.Definition().Name
	})
	return res
}

func (r LocalRegistry) ListRelevant(ctx context.Context, userInput string) []domain.AssistantActionDefinition {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	queryVector, err := r.se.VectorizeQuery(spanCtx, r.embeddingModel, userInput)
	if err != nil || len(queryVector.Vector) == 0 {
		definitions := make([]domain.AssistantActionDefinition, 0, len(r.embeddingByActionName))
		for _, action := range r.embeddingByActionName {
			definitions = append(definitions, action.Action.Definition())
		}
		return definitions
	}

	type scoredAction struct {
		definition domain.AssistantActionDefinition
		score      float64
	}

	scoredActions := make([]scoredAction, 0, len(r.embeddingByActionName))
	for _, actionDetail := range r.embeddingByActionName {
		score, ok := common.CosineSimilarity(queryVector.Vector, actionDetail.Embedding)
		if !ok || score < defaultRelevantActionsMinScore {
			continue
		}

		scoredActions = append(scoredActions, scoredAction{
			definition: actionDetail.Action.Definition(),
			score:      score,
		})
	}

	if len(scoredActions) == 0 {
		definitions := make([]domain.AssistantActionDefinition, 0, len(r.embeddingByActionName))
		for _, action := range r.embeddingByActionName {
			definitions = append(definitions, action.Action.Definition())
		}
		return definitions
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

type InitLocalActionRegistry struct {
	Uow             domain.UnitOfWork          `resolve:""`
	TodoCreator     usecases.TodoCreator       `resolve:""`
	TodoUpdater     usecases.TodoUpdater       `resolve:""`
	TodoDeleter     usecases.TodoDeleter       `resolve:""`
	TodoRepo        domain.TodoRepository      `resolve:""`
	SemanticEncoder domain.SemanticEncoder     `resolve:""`
	TimeProvider    domain.CurrentTimeProvider `resolve:""`
	EmbeddingModel  string                     `config:"LLM_EMBEDDING_MODEL"`
}

func (i InitLocalActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	actions := []domain.AssistantAction{
		actions.NewUIFiltersSetterAction(),
		actions.NewTodoFetcherAction(
			i.TodoRepo,
			i.SemanticEncoder,
			i.EmbeddingModel,
		),
		actions.NewBulkTodoCreatorAction(
			i.Uow,
			i.TodoCreator,
			i.TimeProvider,
		),
		actions.NewBulkTodoUpdaterAction(
			i.Uow,
			i.TodoUpdater,
		),
		actions.NewBulkTodoDueDateUpdaterAction(
			i.Uow,
			i.TodoUpdater,
			i.TimeProvider,
		),
		actions.NewBulkTodoDeleterAction(
			i.Uow,
			i.TodoDeleter,
		),
	}

	actionVectors, err := generateActionVectors(spanCtx, actions, i.SemanticEncoder, i.EmbeddingModel)
	if err != nil {
		return ctx, fmt.Errorf("failed to build assistant action vectors: %w", err)
	}

	actionRegistry := NewActionRegistry(i.SemanticEncoder, i.EmbeddingModel, actionVectors...)
	depend.RegisterNamed[actionregistry.EmbeddingActionRegistry](actionRegistry, "local")
	return ctx, nil
}

// generateActionVectors generates vector embeddings for a list of assistant actions.
func generateActionVectors(ctx context.Context, actions []domain.AssistantAction, encoder domain.SemanticEncoder, embeddingModel string) ([]actionregistry.ActionEmbedding, error) {
	var details []actionregistry.ActionEmbedding
	for _, action := range actions {
		vector, err := encoder.VectorizeAssistantActionDefinition(ctx, embeddingModel, action.Definition())
		if err != nil {
			return nil, fmt.Errorf("failed to vectorize action '%s': %w", action.Definition().Name, err)
		}
		details = append(details, actionregistry.ActionEmbedding{
			Action:    action,
			Embedding: vector.Vector,
		})
	}
	return details, nil
}
