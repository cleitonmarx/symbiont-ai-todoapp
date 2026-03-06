package skillregistry

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInitLocalSkillRegistry_Initialize(t *testing.T) {
	t.Parallel()

	enc := semantic.NewMockEncoder(t)
	enc.EXPECT().
		VectorizeSkillDefinition(mock.Anything, "test-embedding-model", mock.Anything).
		Return(semantic.EmbeddingVector{Vector: []float64{1, 0}}, semantic.EmbeddingVector{Vector: []float64{0, 1}}, nil)

	i := InitLocalSkillRegistry{
		Encoder:        enc,
		EmbeddingModel: "test-embedding-model",
	}

	ctx, err := i.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	dep, err := depend.Resolve[assistant.SkillRegistry]()
	assert.NoError(t, err)
	assert.NotNil(t, dep)

}
