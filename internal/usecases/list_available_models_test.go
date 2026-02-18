package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListAvailableModelsImpl_Query(t *testing.T) {
	tests := map[string]struct {
		setExpectations func(
			assistantCatalog *domain.MockAssistantModelCatalog,
			embeddingCatalog *domain.MockEmbeddingModelCatalog,
		)
		expectedModels []domain.ModelInfo
		expectedErr    error
	}{
		"success": {
			setExpectations: func(assistantCatalog *domain.MockAssistantModelCatalog, embeddingCatalog *domain.MockEmbeddingModelCatalog) {
				assistantCatalog.EXPECT().
					ListAssistantModels(mock.Anything).
					Return([]domain.AssistantModelInfo{
						{Name: "gpt-4"},
					}, nil).
					Once()
				embeddingCatalog.EXPECT().
					ListEmbeddingModels(mock.Anything).
					Return([]domain.EmbeddingModelInfo{
						{Name: "text-embed"},
					}, nil).
					Once()
			},
			expectedModels: []domain.ModelInfo{
				{Name: "gpt-4", Kind: domain.ModelKindAssistant},
				{Name: "text-embed", Kind: domain.ModelKindEmbedding},
			},
			expectedErr: nil,
		},
		"assistant-catalog-error": {
			setExpectations: func(assistantCatalog *domain.MockAssistantModelCatalog, embeddingCatalog *domain.MockEmbeddingModelCatalog) {
				assistantCatalog.EXPECT().
					ListAssistantModels(mock.Anything).
					Return(nil, errors.New("llm error")).
					Once()
			},
			expectedModels: nil,
			expectedErr:    errors.New("llm error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assistantCatalog := domain.NewMockAssistantModelCatalog(t)
			embeddingCatalog := domain.NewMockEmbeddingModelCatalog(t)
			tt.setExpectations(assistantCatalog, embeddingCatalog)

			uc := NewListAvailableModelsImpl(
				assistantCatalog,
				embeddingCatalog,
			)
			got, err := uc.Query(context.Background())

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedModels, got)
		})
	}
}

func TestInitListAvailableModels_Initialize(t *testing.T) {
	assistantCatalog := domain.NewMockAssistantModelCatalog(t)
	embeddingCatalog := domain.NewMockEmbeddingModelCatalog(t)
	init := InitListAvailableModels{
		AssistantCatalog: assistantCatalog,
		EmbeddingCatalog: embeddingCatalog,
	}

	_, err := init.Initialize(context.Background())
	assert.NoError(t, err)

	uc, err := depend.Resolve[ListAvailableModels]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)
}
