package modelrunner

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitAssistantClient_Initialize(t *testing.T) {
	t.Parallel()

	i := InitAssistantClient{}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	r, err := depend.Resolve[assistant.Assistant]()
	assert.NotNil(t, r)
	assert.NoError(t, err)

	catalog, err := depend.Resolve[assistant.ModelCatalog]()
	assert.NotNil(t, catalog)
	assert.NoError(t, err)
}

func TestInitEncoderClient_Initialize(t *testing.T) {
	t.Parallel()

	i := InitEncoderClient{}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	encoder, err := depend.Resolve[semantic.Encoder]()
	assert.NotNil(t, encoder)
	assert.NoError(t, err)
}
