package time

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitCurrentTimeProvider_Initialize(t *testing.T) {
	t.Parallel()

	i := &InitCurrentTimeProvider{}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	_, err = depend.Resolve[core.CurrentTimeProvider]()
	assert.NoError(t, err)
}
