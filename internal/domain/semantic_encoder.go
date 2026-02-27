package domain

import "context"

// EmbeddingVector is a semantic vector plus token accounting.
type EmbeddingVector struct {
	Vector      []float64
	TotalTokens int
}

// SemanticEncoder defines embedding/vectorization behavior in domain terms.
type SemanticEncoder interface {
	// VectorizeTodo generates a semantic vector for one todo item.
	VectorizeTodo(ctx context.Context, model string, todo Todo) (EmbeddingVector, error)
	// VectorizeQuery generates a semantic vector for one user query/search input.
	VectorizeQuery(ctx context.Context, model, query string) (EmbeddingVector, error)
	// VectorizeSkillDefinition generates a semantic vector for one assistant skill definition.
	VectorizeSkillDefinition(ctx context.Context, model string, skill AssistantSkillDefinition) (EmbeddingVector, EmbeddingVector, error)
}
