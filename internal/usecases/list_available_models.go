package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
)

// ListAvailableModels defines the use case for listing available assistant models
type ListAvailableModels interface {
	Query(ctx context.Context) ([]domain.ModelInfo, error)
}

// ListAvailableModelsImpl implements the ListAvailableModels use case
type ListAvailableModelsImpl struct {
	assistantCatalog domain.AssistantModelCatalog
	embeddingCatalog domain.EmbeddingModelCatalog
}

// NewListAvailableModelsImpl creates a new ListAvailableModelsImpl instance
func NewListAvailableModelsImpl(
	assistantCatalog domain.AssistantModelCatalog,
	embeddingCatalog domain.EmbeddingModelCatalog,
) *ListAvailableModelsImpl {
	return &ListAvailableModelsImpl{
		assistantCatalog: assistantCatalog,
		embeddingCatalog: embeddingCatalog,
	}
}

// Query retrieves the list of available assistant and embedding models
func (uc ListAvailableModelsImpl) Query(ctx context.Context) ([]domain.ModelInfo, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	assistantModels, err := uc.assistantCatalog.ListAssistantModels(spanCtx)
	if err != nil {
		return nil, err
	}
	embeddingModels, err := uc.embeddingCatalog.ListEmbeddingModels(spanCtx)
	if err != nil {
		return nil, err
	}

	res := make([]domain.ModelInfo, 0, len(assistantModels)+len(embeddingModels))
	for _, m := range assistantModels {
		res = append(res, domain.ModelInfo{
			Name: m.Name,
			Kind: domain.ModelKindAssistant,
		})
	}
	for _, m := range embeddingModels {
		res = append(res, domain.ModelInfo{
			Name: m.Name,
			Kind: domain.ModelKindEmbedding,
		})
	}
	return res, nil
}

type InitListAvailableModels struct {
	AssistantCatalog domain.AssistantModelCatalog `resolve:""`
	EmbeddingCatalog domain.EmbeddingModelCatalog `resolve:""`
}

// Initialize registers the ListAvailableModels use case in the dependency container
func (i InitListAvailableModels) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListAvailableModels](NewListAvailableModelsImpl(
		i.AssistantCatalog,
		i.EmbeddingCatalog,
	))
	return ctx, nil
}
