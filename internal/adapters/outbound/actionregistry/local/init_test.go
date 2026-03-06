package local

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitActionRegistry_Initialize(t *testing.T) {
	t.Parallel()

	i := InitLocalActionRegistry{}
	registry, err := i.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, registry)

	dependency, err := depend.ResolveNamed[assistant.ActionRegistry]("local")
	assert.NoError(t, err)
	assert.IsType(t, LocalRegistry{}, dependency)
}
