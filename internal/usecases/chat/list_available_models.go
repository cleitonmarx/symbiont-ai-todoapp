package chat

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// ListAvailableModels returns the assistant models exposed to chat clients.
type ListAvailableModels interface {
	// Query returns the currently available assistant models.
	Query(ctx context.Context) ([]assistant.ModelInfo, error)
}

// ListAvailableModelsImpl implements ListAvailableModels.
type ListAvailableModelsImpl struct {
	assistantCatalog assistant.ModelCatalog
}

// NewListAvailableModelsImpl creates a ListAvailableModelsImpl.
func NewListAvailableModelsImpl(
	assistantCatalog assistant.ModelCatalog,
) *ListAvailableModelsImpl {
	return &ListAvailableModelsImpl{
		assistantCatalog: assistantCatalog,
	}
}

// Query implements ListAvailableModels.
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
