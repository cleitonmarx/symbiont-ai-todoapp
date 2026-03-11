package chat

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// ListAvailableModels defines the use case for listing available assistant models
type ListAvailableModels interface {
	Query(ctx context.Context) ([]assistant.ModelInfo, error)
}

// ListAvailableModelsImpl implements the ListAvailableModels use case
type ListAvailableModelsImpl struct {
	assistantCatalog assistant.ModelCatalog
}

// NewListAvailableModelsImpl creates a new ListAvailableModelsImpl instance
func NewListAvailableModelsImpl(
	assistantCatalog assistant.ModelCatalog,
) *ListAvailableModelsImpl {
	return &ListAvailableModelsImpl{
		assistantCatalog: assistantCatalog,
	}
}

// Query retrieves the list of available assistant models
func (uc ListAvailableModelsImpl) Query(ctx context.Context) ([]assistant.ModelInfo, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	assistantModels, err := uc.assistantCatalog.ListModels(spanCtx)
	if telemetry.IsErrorRecorded(span, err) {
		return nil, err
	}

	res := make([]assistant.ModelInfo, 0, len(assistantModels))
	for _, m := range assistantModels {
		res = append(res, assistant.ModelInfo{
			ID:   m.ID,
			Name: m.Name,
			Kind: assistant.ModelKindAssistant,
		})
	}
	return res, nil
}
