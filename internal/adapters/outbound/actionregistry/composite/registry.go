package composite

import (
	"context"
	"fmt"
	"sort"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
)

const (
	defaultRelevantActionsTopK     = 3
	defaultRelevantActionsMinScore = 0.35
)

// CompositeActionRegistry implements domain.AssistantActionRegistry interface.
// It aggregates actions from multiple EmbeddingActionRegistry instances.
type CompositeActionRegistry struct {
	se                domain.SemanticEncoder
	embeddingModel    string
	registriesActions map[string]actionregistry.ActionEmbedding
}

// NewCompositeActionRegistry creates a new CompositeActionRegistry from the given embedding registries.
func NewCompositeActionRegistry(ctx context.Context, se domain.SemanticEncoder, embeddingModel string, registries ...actionregistry.EmbeddingActionRegistry) CompositeActionRegistry {
	registryMap := make(map[string]actionregistry.ActionEmbedding)
	for _, registry := range registries {
		actions := registry.ListEmbeddings(ctx)
		for _, action := range actions {
			registryMap[action.Action.Definition().Name] = action
		}
	}

	return CompositeActionRegistry{
		se:                se,
		embeddingModel:    embeddingModel,
		registriesActions: registryMap,
	}
}

// Execute iterates through the composed registries to execute the given action call, returning the first successful result.
func (r CompositeActionRegistry) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	actionEmbedding, found := r.registriesActions[call.Name]
	if !found {
		return domain.AssistantMessage{
			Role:    domain.ChatRole_Tool,
			Content: fmt.Sprintf("error: no registry found for action '%s'", call.Name),
		}
	}

	return actionEmbedding.Action.Execute(spanCtx, call, conversationHistory)
}

// StatusMessage iterates through the composed registries to get the status message for the given action, returning a default message if none found.
func (r CompositeActionRegistry) StatusMessage(actionName string) string {
	actionEmbedding, found := r.registriesActions[actionName]
	if !found {
		return "⏳ Processing request..."
	}
	return actionEmbedding.Action.StatusMessage()
}

// ListRelevant aggregates and deduplicates relevant action definitions from all composed registries based on the user input, sorted by name.
func (r CompositeActionRegistry) ListRelevant(ctx context.Context, userInput string) []domain.AssistantActionDefinition {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	queryVector, err := r.se.VectorizeQuery(spanCtx, r.embeddingModel, userInput)
	if err != nil || len(queryVector.Vector) == 0 {
		definitions := make([]domain.AssistantActionDefinition, 0, len(r.registriesActions))
		for _, action := range r.registriesActions {
			definitions = append(definitions, action.Action.Definition())
		}
		return definitions
	}

	type scoredAction struct {
		definition domain.AssistantActionDefinition
		score      float64
	}

	scoredActions := make([]scoredAction, 0, len(r.registriesActions))
	for _, a := range r.registriesActions {
		score, ok := common.CosineSimilarity(queryVector.Vector, a.Embedding)
		if !ok || score < defaultRelevantActionsMinScore {
			continue
		}

		scoredActions = append(scoredActions, scoredAction{
			definition: a.Action.Definition(),
			score:      score,
		})
	}

	if len(scoredActions) == 0 {
		definitions := make([]domain.AssistantActionDefinition, 0, len(r.registriesActions))
		for _, action := range r.registriesActions {
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

// InitCompositeActionRegistry is the initializer for CompositeActionRegistry, composing local and MCP gateway registries.
type InitCompositeActionRegistry struct {
	Local           actionregistry.EmbeddingActionRegistry `resolve:"local"`
	MCP             actionregistry.EmbeddingActionRegistry `resolve:"mcp"`
	SemanticEncoder domain.SemanticEncoder                 `resolve:""`
	EmbeddingModel  string                                 `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize creates a CompositeActionRegistry from the local and MCP gateway registries and registers it in the dependency container.
func (i InitCompositeActionRegistry) Initialize(ctx context.Context) (context.Context, error) {
	composite := NewCompositeActionRegistry(ctx, i.SemanticEncoder, i.EmbeddingModel, i.Local, i.MCP)
	depend.Register[domain.AssistantActionRegistry](composite)
	return ctx, nil
}
