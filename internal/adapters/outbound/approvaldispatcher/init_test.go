package approvaldispatcher

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitDispatcher_Initialize(t *testing.T) {
	i := InitDispatcher{}

	ctx, err := i.Initialize(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, ctx)

	registered, err := depend.Resolve[assistant.ActionApprovalDispatcher]()
	require.NoError(t, err)
	assert.NotNil(t, registered)
}
