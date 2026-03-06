package skillregistry

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/stretchr/testify/mock"
)

type semanticEncoderParams struct {
	QueryVectors map[string][]float64
	QueryErrors  map[string]error
	SkillVectors map[string]skillVector
	SkillErrors  map[string]error
}

type skillVector struct {
	Use   []float64
	Avoid []float64
}

func newSemanticEncoder(t *testing.T, model string, params semanticEncoderParams) *semantic.MockEncoder {
	t.Helper()

	enc := semantic.NewMockEncoder(t)
	enc.EXPECT().
		VectorizeQuery(mock.Anything, model, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, query string) (semantic.EmbeddingVector, error) {
			if err, ok := params.QueryErrors[query]; ok {
				return semantic.EmbeddingVector{}, err
			}
			if vec, ok := params.QueryVectors[query]; ok {
				return semantic.EmbeddingVector{Vector: vec}, nil
			}
			return semantic.EmbeddingVector{}, nil
		})

	for name, vector := range params.SkillVectors {
		skillName := name
		enc.EXPECT().
			VectorizeSkillDefinition(
				mock.Anything,
				model,
				mock.MatchedBy(func(skill assistant.SkillDefinition) bool { return skill.Name == skillName }),
			).
			Return(
				semantic.EmbeddingVector{Vector: vector.Use},
				semantic.EmbeddingVector{Vector: vector.Avoid},
				nil,
			).
			Once()
	}
	for name, err := range params.SkillErrors {
		skillName := name
		enc.EXPECT().
			VectorizeSkillDefinition(
				mock.Anything,
				model,
				mock.MatchedBy(func(skill assistant.SkillDefinition) bool { return skill.Name == skillName }),
			).
			Return(semantic.EmbeddingVector{}, semantic.EmbeddingVector{}, err).
			Once()
	}

	return enc
}
