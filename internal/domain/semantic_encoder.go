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
	// VectorizeAssistantActionDefinition generates a semantic vector for one assistant action definition.
	VectorizeAssistantActionDefinition(ctx context.Context, model string, action AssistantActionDefinition) (EmbeddingVector, error)
}

// EmbeddingModelInfo describes a model usable for vectorization.
type EmbeddingModelInfo struct {
	Name string
}

// EmbeddingModelCatalog exposes available embedding-capable models.
type EmbeddingModelCatalog interface {
	ListEmbeddingModels(ctx context.Context) ([]EmbeddingModelInfo, error)
}
