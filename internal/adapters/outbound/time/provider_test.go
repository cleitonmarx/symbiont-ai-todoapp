package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCurrentTimeProvider_Now(t *testing.T) {
	t.Parallel()

	p := CurrentTimeProvider{}
	now := p.Now()
	assert.WithinDuration(t, time.Now(), now, time.Second)
}
