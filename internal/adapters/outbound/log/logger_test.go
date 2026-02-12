package log

import (
	"context"
	"log"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitLogger_Initialize(t *testing.T) {
	init := InitLogger{}

	_, err := init.Initialize(context.Background())
	assert.NoError(t, err)

	_, err = depend.Resolve[*log.Logger]()
	assert.NoError(t, err)

}
