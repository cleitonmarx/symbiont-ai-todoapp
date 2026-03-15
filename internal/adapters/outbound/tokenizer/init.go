package tokenizer

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitTokenizer registers the default tokenizer implementation.
type InitTokenizer struct{}

// Initialize registers the tokenizer in the dependency container.
func (it InitTokenizer) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[assistant.Tokenizer](DefaultTokenizer{})
	return ctx, nil
}
