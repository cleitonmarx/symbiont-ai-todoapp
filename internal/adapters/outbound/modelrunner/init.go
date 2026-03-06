package modelrunner

import (
	"context"
	"net/http"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitAssistantClient initializes the assistant client dependency.
type InitAssistantClient struct {
	HttpClient         *http.Client `resolve:""`
	EmbeddingModelHost string       `config:"LLM_EMBEDDING_MODEL_HOST"`
	ModelHost          string       `config:"LLM_MODEL_HOST"`
	APIKey             string       `config:"LLM_API_KEY" default:""`
	EmbeddingAPIKey    string       `config:"LLM_EMBEDDING_API_KEY" default:""`
}

// Initialize creates and registers the assistant client and related interfaces in the dependency container.
func (i InitAssistantClient) Initialize(ctx context.Context) (context.Context, error) {
	adapter := NewAssistantClientAdapter(
		NewDRMAPIClient(i.ModelHost, i.APIKey, i.HttpClient),
		NewDRMAPIClient(i.EmbeddingModelHost, i.EmbeddingAPIKey, i.HttpClient),
	)
	depend.Register[assistant.Assistant](adapter)
	depend.Register[semantic.Encoder](adapter)
	depend.Register[assistant.ModelCatalog](adapter)
	return ctx, nil
}
