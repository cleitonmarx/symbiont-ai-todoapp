package skillregistry

import (
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

	seenQueries := make(map[string]struct{}, len(params.QueryErrors)+len(params.QueryVectors))
	for query, err := range params.QueryErrors {
		seenQueries[query] = struct{}{}
		enc.EXPECT().
			VectorizeQuery(mock.Anything, model, query).
			Return(semantic.EmbeddingVector{}, err).
			Once()
	}
	for query, vec := range params.QueryVectors {
		if _, alreadySetAsError := seenQueries[query]; alreadySetAsError {
			continue
		}
		enc.EXPECT().
			VectorizeQuery(mock.Anything, model, query).
			Return(semantic.EmbeddingVector{Vector: vec}, nil).
			Once()
	}

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
