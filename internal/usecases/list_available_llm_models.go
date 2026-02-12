package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
)

// ListAvailableLLMModels defines the use case for listing available LLM models
type ListAvailableLLMModels interface {
	Query(ctx context.Context) ([]domain.LLMModelInfo, error)
}

// ListAvailableLLMModelsImpl implements the ListAvailableLLMModels use case
type ListAvailableLLMModelsImpl struct {
	llmClient domain.LLMClient
}

// NewListAvailableLLMModelsImpl creates a new ListAvailableLLMModelsImpl instance
func NewListAvailableLLMModelsImpl(llmClient domain.LLMClient) *ListAvailableLLMModelsImpl {
	return &ListAvailableLLMModelsImpl{
		llmClient: llmClient,
	}
}

// Query retrieves the list of available LLM models
func (uc ListAvailableLLMModelsImpl) Query(ctx context.Context) ([]domain.LLMModelInfo, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	return uc.llmClient.AvailableModels(spanCtx)
}

type InitListAvailableLLMModels struct {
	LLMClient domain.LLMClient `resolve:""`
}

// Initialize registers the ListAvailableLLMModels use case in the dependency container
func (i InitListAvailableLLMModels) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListAvailableLLMModels](NewListAvailableLLMModelsImpl(i.LLMClient))
	return ctx, nil
}
