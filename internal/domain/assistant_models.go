package domain

import "context"

// ModelKind describes the capability class of a model.
type ModelKind string

const (
	ModelKindAssistant ModelKind = "assistant"
	ModelKindEmbedding ModelKind = "embedding"
)

// ModelInfo describes one available model in a provider-agnostic format.
type ModelInfo struct {
	Name string
	Kind ModelKind
}

// AssistantModelInfo describes a model that can be used for assistant turns.
type AssistantModelInfo struct {
	Name string
	// SupportsStreaming indicates the model can emit incremental deltas.
	SupportsStreaming bool
	// SupportsActions indicates the model can request assistant actions/tools.
	SupportsActions bool
}

// AssistantModelCatalog exposes available assistant-capable models.
type AssistantModelCatalog interface {
	ListAssistantModels(ctx context.Context) ([]AssistantModelInfo, error)
}
