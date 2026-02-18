package domain

import "context"

// EmbedResponse represents the response from an embedding request to the LLM API.
type EmbedResponse struct {
	Embedding   []float64
	TotalTokens int
}

type LLMModelType string

const (
	LLMModelType_Chat      LLMModelType = "chat"
	LLMModelType_Embedding LLMModelType = "embedding"
)

// LLMModelInfo represents information about an available LLM model.
type LLMModelInfo struct {
	Name string
	Type LLMModelType
}

// LLMClient defines the interface for interacting with an LLM API.
type LLMClient interface {
	// ChatStream streams assistant output as events from an LLM server.
	// It calls onEvent with each event (meta, delta, tool_call/tool_call_started/tool_call_finished, done) and returns any error.
	ChatStream(ctx context.Context, req LLMChatRequest, onEvent LLMStreamEventCallback) error

	// Chat sends a chat request to the LLM and returns the full assistant response.
	Chat(ctx context.Context, req LLMChatRequest) (LLMChatResponse, error)

	// EmbedTodo creates an embedding for the given todo item, used for indexing in the vector database.
	EmbedTodo(ctx context.Context, model string, todo Todo) (EmbedResponse, error)

	// EmbedSearch creates an embedding for the given input string, used for similarity search in the vector database.
	EmbedSearch(ctx context.Context, model, searchInput string) (EmbedResponse, error)

	// AvailableModels retrieves the list of available models.
	AvailableModels(ctx context.Context) ([]LLMModelInfo, error)
}
