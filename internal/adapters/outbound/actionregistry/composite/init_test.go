package composite

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCompositeActionRegistry_Initialize(t *testing.T) {
	t.Parallel()

	r := &InitCompositeActionRegistry{}
	ctx, err := r.Initialize(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, ctx)
	dep, err := depend.Resolve[assistant.ActionRegistry]()
	require.NoError(t, err)
	assert.IsType(t, CompositeActionRegistry{}, dep)
}
