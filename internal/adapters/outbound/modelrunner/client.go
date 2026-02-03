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

	httpReq, err := c.newPostRequest(ctx, "/v1/chat/completions", req)
	if err != nil {
		return nil, err
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

	if resp.StatusCode != http.StatusOK {
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

	req.Stream = true

	httpReq, err := c.newPostRequest(ctx, "/v1/chat/completions", req)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
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

// Embeddings calls the /engines/v1/embeddings endpoint.
func (c DRMAPIClient) Embeddings(ctx context.Context, req EmbeddingsRequest) (*EmbeddingsResponse, error) {
	httpReq, err := c.newPostRequest(ctx, "/engines/v1/embeddings", req)
	if err != nil {
		return nil, err
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

	var out EmbeddingsResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &out, nil
}

func (c DRMAPIClient) newPostRequest(ctx context.Context, path string, body any) (*http.Request, error) {
	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	return req, nil
}
