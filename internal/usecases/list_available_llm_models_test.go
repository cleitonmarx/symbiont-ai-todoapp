package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListAvailableLLMModelsImpl_Query(t *testing.T) {
	tests := map[string]struct {
		setExpectations func(llm *domain.MockLLMClient)
		expectedModels  []domain.LLMModelInfo
		expectedErr     error
	}{
		"success": {
			setExpectations: func(llm *domain.MockLLMClient) {
				llm.EXPECT().
					AvailableModels(mock.Anything).
					Return([]domain.LLMModelInfo{
						{Name: "gpt-4", Type: domain.LLMModelType_Chat},
						{Name: "text-embed", Type: domain.LLMModelType_Embedding},
					}, nil).
					Once()
			},
			expectedModels: []domain.LLMModelInfo{
				{Name: "gpt-4", Type: domain.LLMModelType_Chat},
				{Name: "text-embed", Type: domain.LLMModelType_Embedding},
			},
			expectedErr: nil,
		},
		"llm-client-error": {
			setExpectations: func(llm *domain.MockLLMClient) {
				llm.EXPECT().
					AvailableModels(mock.Anything).
					Return(nil, errors.New("llm error")).
					Once()
			},
			expectedModels: nil,
			expectedErr:    errors.New("llm error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			llm := domain.NewMockLLMClient(t)
			tt.setExpectations(llm)

			uc := NewListAvailableLLMModelsImpl(llm)
			got, err := uc.Query(context.Background())

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedModels, got)
		})
	}
}

func TestInitListAvailableLLMModels_Initialize(t *testing.T) {
	init := InitListAvailableLLMModels{
		LLMClient: domain.NewMockLLMClient(t),
	}

	_, err := init.Initialize(context.Background())
	assert.NoError(t, err)

	uc, err := depend.Resolve[ListAvailableLLMModels]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)
}
