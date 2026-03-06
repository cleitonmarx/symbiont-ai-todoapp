package board

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitGenerateBoardSummary_Initialize(t *testing.T) {
	t.Parallel()

	igbs := InitGenerateBoardSummary{}

	ctx, err := igbs.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredGbs, err := depend.Resolve[GenerateBoardSummary]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredGbs)
}

func TestInitGetBoardSummary_Initialize(t *testing.T) {
	t.Parallel()

	summaryRepo := todo.NewMockBoardSummaryRepository(t)

	igbs := &InitGetBoardSummary{
		SummaryRepo: summaryRepo,
	}

	ctx, err := igbs.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
}
