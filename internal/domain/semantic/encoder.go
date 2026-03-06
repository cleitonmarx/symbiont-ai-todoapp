package semantic

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
)

// EmbeddingVector is a semantic vector plus token accounting.
type EmbeddingVector struct {
	Vector      []float64
	TotalTokens int
}

// Encoder defines embedding/vectorization behavior in domain terms.
type Encoder interface {
	// VectorizeTodo generates a semantic vector for one todo item.
	VectorizeTodo(ctx context.Context, model string, todo todo.Todo) (EmbeddingVector, error)
	// VectorizeQuery generates a semantic vector for one user query/search input.
	VectorizeQuery(ctx context.Context, model, query string) (EmbeddingVector, error)
	// VectorizeSkillDefinition generates semantic vectors for one assistant skill definition.
	VectorizeSkillDefinition(ctx context.Context, model string, skill assistant.SkillDefinition) (EmbeddingVector, EmbeddingVector, error)
}
