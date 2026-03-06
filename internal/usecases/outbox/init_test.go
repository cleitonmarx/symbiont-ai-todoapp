package outbox

import (
	"context"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitRelayOutbox_Initialize(t *testing.T) {
	t.Parallel()

	iro := InitRelay{}

	ctx, err := iro.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredRelay, err := depend.Resolve[Relay]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredRelay)
}
