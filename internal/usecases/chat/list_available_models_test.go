package chat

import (
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListAvailableModelsImpl_Query(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setExpectations func(*assistant.MockModelCatalog)
		expectedModels  []assistant.ModelInfo
		expectedErr     error
	}{
		"success": {
			setExpectations: func(assistantCatalog *assistant.MockModelCatalog) {
				assistantCatalog.EXPECT().
					ListModels(mock.Anything).
					Return([]assistant.ModelCapabilities{
						{ID: "gpt-4", Name: "gpt-4"},
					}, nil).
					Once()
			},
			expectedModels: []assistant.ModelInfo{
				{ID: "gpt-4", Name: "gpt-4", Kind: assistant.ModelKindAssistant},
			},
			expectedErr: nil,
		},
		"assistant-catalog-error": {
			setExpectations: func(assistantCatalog *assistant.MockModelCatalog) {
				assistantCatalog.EXPECT().
					ListModels(mock.Anything).
					Return(nil, errors.New("llm error")).
					Once()
			},
			expectedModels: nil,
			expectedErr:    errors.New("llm error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assistantCatalog := assistant.NewMockModelCatalog(t)
			tt.setExpectations(assistantCatalog)

			uc := NewListAvailableModelsImpl(
				assistantCatalog,
			)
			got, err := uc.Query(t.Context())

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedModels, got)
		})
	}
}
