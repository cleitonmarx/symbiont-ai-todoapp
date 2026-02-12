package time

import (
	"context"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestInitCurrentTimeProvider_Initialize(t *testing.T) {
	i := &InitCurrentTimeProvider{}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	_, err = depend.Resolve[domain.CurrentTimeProvider]()
	assert.NoError(t, err)
}

func TestCurrentTimeProvider_Now(t *testing.T) {
	p := CurrentTimeProvider{}
	now := p.Now()
	assert.WithinDuration(t, time.Now(), now, time.Second)
}
