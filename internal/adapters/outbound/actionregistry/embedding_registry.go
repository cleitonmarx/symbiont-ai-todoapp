package actionregistry

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// ActionEmbedding holds an assistant action and its corresponding vector embedding for relevance scoring.
type ActionEmbedding struct {
	Action    domain.AssistantAction
	Embedding []float64
}

// EmbeddingActionRegistry extends AssistantActionRegistry to include vector embeddings for actions, enabling relevance scoring.
type EmbeddingActionRegistry interface {
	domain.AssistantActionRegistry
	// ListEmbeddings returns all available assistant action along with their vector embeddings for relevance scoring.
	ListEmbeddings(ctx context.Context) []ActionEmbedding
}
