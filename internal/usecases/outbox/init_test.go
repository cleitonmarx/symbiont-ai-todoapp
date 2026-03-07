package outbox

import (
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitRelayOutbox_Initialize(t *testing.T) {
	t.Parallel()

	iro := InitRelay{}

	ctx, err := iro.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredRelay, err := depend.Resolve[Relay]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredRelay)
}
