package modelrunner

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitAssistantClient_Initialize(t *testing.T) {
	t.Parallel()

	i := InitAssistantClient{}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	r, err := depend.Resolve[assistant.Assistant]()
	assert.NotNil(t, r)
	assert.NoError(t, err)
}
