package assistant

import "context"

// ModelKind describes the capability class of a model.
type ModelKind string

const (
	// ModelKindAssistant identifies assistant/chat-capable models.
	ModelKindAssistant ModelKind = "assistant"
	// ModelKindEmbedding identifies embedding-capable models.
	ModelKindEmbedding ModelKind = "embedding"
)

// ModelInfo describes one available model in a provider-agnostic format.
type ModelInfo struct {
	ID   string
	Name string
	Kind ModelKind
}

// ModelCapabilities describes a model that can be used for assistant turns.
type ModelCapabilities struct {
	ID   string
	Name string
	// SupportsStreaming indicates the model can emit incremental deltas.
	SupportsStreaming bool
	// SupportsActions indicates the model can request assistant actions/tools.
	SupportsActions bool
}

// ModelCatalog exposes available assistant-capable models.
type ModelCatalog interface {
	ListModels(ctx context.Context) ([]ModelCapabilities, error)
}
