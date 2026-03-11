package log

import (
	"log"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitLogger_Initialize(t *testing.T) {
	t.Parallel()

	init := InitLogger{}

	_, err := init.Initialize(t.Context())
	assert.NoError(t, err)

	_, err = depend.Resolve[*log.Logger]()
	assert.NoError(t, err)

}
