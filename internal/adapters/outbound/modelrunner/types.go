package modelrunner

// ChatRequest is an OpenAI-compatible chat completions request
type ChatRequest struct {
	Model         string         `json:"model"`
	Messages      []ChatMessage  `json:"messages"`
	Stream        bool           `json:"stream,omitempty"`
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
	Temperature   *float64       `json:"temperature,omitempty"`
	MaxTokens     *int           `json:"max_tokens,omitempty"`
	TopP          *float64       `json:"top_p,omitempty"`
	Tools         []Tool         `json:"tools,omitempty"`
}

// StreamOptions represents options for streaming responses
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// Tool represents a tool the model may call (OpenAI function tool)
type Tool struct {
	Type     string   `json:"type"`
	Function ToolFunc `json:"function"`
}

// ToolFunc represents the function object in a tool
type ToolFunc struct {
	Description string             `json:"description"`
	Name        string             `json:"name"`
	Parameters  ToolFuncParameters `json:"parameters"`
	Required    []string           `json:"required,omitempty"`
}

// ToolFuncParameters represents the parameters schema for a function tool (OpenAI JSON Schema)
type ToolFuncParameters struct {
	Type                 string                             `json:"type"`
	Properties           map[string]ToolFuncParameterDetail `json:"properties"`
	AdditionalProperties bool                               `json:"additionalProperties"`
}

// ToolFuncParameterDetail represents a single parameter in the function tool schema
type ToolFuncParameterDetail struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// ChatMessage is an OpenAI-compatible message
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID *string    `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ChatResponse is an OpenAI-compatible response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage"`
	Timings *Timings `json:"timings,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
}

// Message represents the assistant message
type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call made by the model
type ToolCall struct {
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
	ID       string           `json:"id"`
	Index    int              `json:"index,omitempty"`
}

// ToolCallFunction represents the function call details
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

// StreamChunk represents a streaming response chunk from llama.cpp
type StreamChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []StreamChunkChoice `json:"choices"`
	Usage   *Usage              `json:"usage,omitempty"`
	Timings *Timings            `json:"timings,omitempty"`
}

// StreamChunkChoice represents a choice in a streaming chunk
type StreamChunkChoice struct {
	Index        int              `json:"index"`
	FinishReason *string          `json:"finish_reason"`
	Delta        StreamChunkDelta `json:"delta"`
}

// StreamChunkDelta represents the delta content
type StreamChunkDelta struct {
	Role      *string         `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []ToolCallChunk `json:"tool_calls,omitempty"`
}

// ToolCallChunk represents a tool call in a streaming chunk
type ToolCallChunk struct {
	Type     string                `json:"type"`
	Function ToolCallChunkFunction `json:"function"`
	ID       string                `json:"id"`
	Index    int                   `json:"index"`
}

// ToolCallChunkFunction represents the function call details in a streaming chunk
type ToolCallChunkFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Timings contains llama.cpp performance metrics
type Timings struct {
	PromptN    int `json:"prompt_n"`
	PredictedN int `json:"predicted_n"`
}

// EmbeddingsRequest represents the request payload for the embeddings endpoint.
type EmbeddingsRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"` // string or []string
}

// EmbeddingsUsage represents the token usage for embeddings
type EmbeddingsUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingData represents a single embedding
type EmbeddingData struct {
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
	Object    string    `json:"object"`
}

// EmbeddingsResponse represents the response from the embeddings endpoint.
type EmbeddingsResponse struct {
	Model  string          `json:"model"`
	Object string          `json:"object"`
	Usage  EmbeddingsUsage `json:"usage"`
	Data   []EmbeddingData `json:"data"`
}
