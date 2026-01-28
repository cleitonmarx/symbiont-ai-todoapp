// Package llm provides a small, backend-agnostic client for a Docker-hosted
// OpenAI-compatible chat-completions endpoint (e.g. llama.cpp server).
//
// It intentionally ignores any non-standard fields such as "reasoning_content"
// and returns the assistant "content" (which may itself be JSON if you prompt
// the model to output JSON).
package modelrunner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DRMAPIClient is a thin client for llama.cpp OpenAI-compatible API
type DRMAPIClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewDRMAPIClient creates a new client
func NewDRMAPIClient(baseURL string, apiKey string, httpClient *http.Client) DRMAPIClient {
	return DRMAPIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    httpClient,
	}
}

// ChatRequest is an OpenAI-compatible chat completions request
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
}

// ChatMessage is an OpenAI-compatible message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse is an OpenAI-compatible response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
}

// Message represents the assistant message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
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
	Role    *string `json:"role,omitempty"`
	Content string  `json:"content"`
}

// Timings contains llama.cpp performance metrics
type Timings struct {
	PromptN    int `json:"prompt_n"`
	PredictedN int `json:"predicted_n"`
}

// ChunkCallback is called for each streaming chunk
type ChunkCallback func(chunk StreamChunk) error

// Chat sends a non-streaming request
func (c DRMAPIClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		return nil, errors.New("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, errors.New("messages are required")
	}

	endpoint, err := url.JoinPath(c.baseURL, "/v1/chat/completions")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-2xx response: %s: %s", resp.Status, string(respBody))
	}

	var out ChatResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &out, nil
}

// ChatStream streams the response, calling onChunk for each SSE data packet
func (c DRMAPIClient) ChatStream(ctx context.Context, req ChatRequest, onChunk ChunkCallback) error {
	if req.Model == "" {
		return errors.New("model is required")
	}
	if len(req.Messages) == 0 {
		return errors.New("messages are required")
	}

	endpoint, err := url.JoinPath(c.baseURL, "/v1/chat/completions")
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("non-2xx response: %s: %s", resp.Status, string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue // Skip malformed chunks
		}

		if err := onChunk(chunk); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// EmbeddingsRequest represents the request payload for the embeddings endpoint.
type EmbeddingsRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"` // string or []string
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

// Embeddings calls the /engines/v1/embeddings endpoint.
func (c DRMAPIClient) Embeddings(ctx context.Context, req EmbeddingsRequest) (*EmbeddingsResponse, error) {
	endpoint, err := url.JoinPath(c.baseURL, "/engines/v1/embeddings")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-2xx response: %s: %s", resp.Status, string(respBody))
	}

	var out EmbeddingsResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &out, nil
}
