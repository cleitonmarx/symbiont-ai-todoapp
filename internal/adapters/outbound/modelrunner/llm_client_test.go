package modelrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
)

// createStreamingServer creates a test server that sends OpenAI-style streaming chunks
func createStreamingServer(chunks []StreamChunk) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)
		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data) //nolint:errcheck
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n") //nolint:errcheck
		flusher.Flush()
	}))
}

// collectStreamEvents collects all events from a stream
func collectStreamEvents(adapter LLMClient, req domain.LLMChatRequest) ([]domain.LLMStreamEventType, []string, *domain.LLMStreamEventDone, error) {
	var eventTypes []domain.LLMStreamEventType
	var deltaTexts []string
	var doneEvent *domain.LLMStreamEventDone

	err := adapter.ChatStream(context.Background(), req, func(eventType domain.LLMStreamEventType, data interface{}) error {
		eventTypes = append(eventTypes, eventType)

		switch eventType {
		case domain.LLMStreamEventType_Delta:
			delta := data.(domain.LLMStreamEventDelta)
			deltaTexts = append(deltaTexts, delta.Text)
		case domain.LLMStreamEventType_Done:
			done := data.(domain.LLMStreamEventDone)
			doneEvent = &done
		}
		return nil
	})

	return eventTypes, deltaTexts, doneEvent, err
}

func TestLLMClientAdapter_ChatStream(t *testing.T) {
	tests := map[string]struct {
		chunks          []StreamChunk
		expectErr       bool
		expectedEvents  []domain.LLMStreamEventType
		expectedContent string
		validateUsage   func(*testing.T, *domain.LLMStreamEventDone)
	}{
		"success-complete-stream": {
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "Hello world"}}}},
				{Usage: &Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}},
			},
			expectedEvents:  []domain.LLMStreamEventType{"meta", "delta", "done"},
			expectedContent: "Hello world",
			validateUsage: func(t *testing.T, done *domain.LLMStreamEventDone) {
				assert.NotNil(t, done.Usage)
				assert.Equal(t, 10, done.Usage.PromptTokens)
				assert.Equal(t, 5, done.Usage.CompletionTokens)
				assert.Equal(t, 15, done.Usage.TotalTokens)
			},
		},
		"multiple-deltas": {
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "Hello"}}}},
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: " "}}}},
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "world"}}}},
			},
			expectedEvents:  []domain.LLMStreamEventType{"meta", "delta", "delta", "delta", "done"},
			expectedContent: "Hello world",
		},
		"usage-from-timings": {
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "test"}}}},
				{Timings: &Timings{PromptN: 8, PredictedN: 4}},
			},
			expectedEvents:  []domain.LLMStreamEventType{"meta", "delta", "done"},
			expectedContent: "test",
			validateUsage: func(t *testing.T, done *domain.LLMStreamEventDone) {
				assert.NotNil(t, done.Usage)
				assert.Equal(t, 8, done.Usage.PromptTokens)
				assert.Equal(t, 4, done.Usage.CompletionTokens)
				assert.Equal(t, 12, done.Usage.TotalTokens)
			},
		},
		"empty-delta": {
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: ""}}}},
			},
			expectedEvents:  []domain.LLMStreamEventType{"meta", "done"},
			expectedContent: "",
		},
		"no-usage-fallback": {
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "test"}}}},
			},
			expectedEvents:  []domain.LLMStreamEventType{"meta", "delta", "done"},
			expectedContent: "test",
			validateUsage: func(t *testing.T, done *domain.LLMStreamEventDone) {
				assert.NotNil(t, done.Usage)
				assert.Greater(t, done.Usage.PromptTokens, 0) // Estimated from message
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := createStreamingServer(tt.chunks)
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewLLMClientAdapter(client)

			req := domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{Role: "user", Content: "test"},
				},
			}

			eventTypes, deltaTexts, doneEvent, err := collectStreamEvents(adapter, req)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, eventTypes)

			combined := strings.Join(deltaTexts, "")
			assert.Equal(t, tt.expectedContent, combined)

			if tt.validateUsage != nil && doneEvent != nil {
				tt.validateUsage(t, doneEvent)
			}
		})
	}
}

func TestLLMClientAdapter_ChatStream_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewDRMAPIClient(server.URL, "", server.Client())
	adapter := NewLLMClientAdapter(client)

	req := domain.LLMChatRequest{
		Model: "test-model",
		Messages: []domain.LLMChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	err := adapter.ChatStream(context.Background(), req, func(eventType domain.LLMStreamEventType, data interface{}) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestLLMClientAdapter_Chat(t *testing.T) {
	temp := 0.5
	topP := 0.9

	tests := map[string]struct {
		response     string
		statusCode   int
		req          domain.LLMChatRequest
		expectErr    bool
		expectedResp string
		validateReq  func(*testing.T, *ChatRequest)
	}{
		"success": {
			response:   `{"choices":[{"message":{"role":"assistant","content":"Hello!"}}]}`,
			statusCode: http.StatusOK,
			req: domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectedResp: "Hello!",
		},
		"with-params": {
			response:   `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`,
			statusCode: http.StatusOK,
			req: domain.LLMChatRequest{
				Model:       "test-model",
				Temperature: &temp,
				TopP:        &topP,
				Messages: []domain.LLMChatMessage{
					{Role: "system", Content: "sys"},
					{Role: "user", Content: "hi"},
				},
			},
			expectedResp: "ok",
			validateReq: func(t *testing.T, req *ChatRequest) {
				assert.Equal(t, "test-model", req.Model)
				assert.NotNil(t, req.Temperature)
				assert.InDelta(t, 0.5, *req.Temperature, 1e-6)
				assert.NotNil(t, req.TopP)
				assert.InDelta(t, 0.9, *req.TopP, 1e-6)
				assert.Len(t, req.Messages, 2)
			},
		},
		"no-choices": {
			response:   `{"choices":[]}`,
			statusCode: http.StatusOK,
			req: domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
		"server-error": {
			response:   `Internal Server Error`,
			statusCode: http.StatusInternalServerError,
			req: domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
		"invalid-json": {
			response:   `{invalid json}`,
			statusCode: http.StatusOK,
			req: domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var capturedReq *ChatRequest

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.validateReq != nil {
					var req ChatRequest
					json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
					capturedReq = &req
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response)) //nolint:errcheck
			}))
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewLLMClientAdapter(client)

			resp, err := adapter.Chat(context.Background(), tt.req)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResp, resp)

			if tt.validateReq != nil && capturedReq != nil {
				tt.validateReq(t, capturedReq)
			}
		})
	}
}

func TestLLMClientAdapter_Chat_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`)) //nolint:errcheck
	}))
	defer server.Close()

	client := NewDRMAPIClient(server.URL, "", server.Client())
	adapter := NewLLMClientAdapter(client)

	tests := map[string]struct {
		req domain.LLMChatRequest
	}{
		"no-model":    {req: domain.LLMChatRequest{Messages: []domain.LLMChatMessage{{Role: "user", Content: "hi"}}}},
		"no-messages": {req: domain.LLMChatRequest{Model: "test"}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := adapter.Chat(context.Background(), tt.req)
			assert.Error(t, err)
		})
	}
}

func TestLLMClientAdapter_Embed(t *testing.T) {
	tests := map[string]struct {
		response    string
		statusCode  int
		model       string
		input       string
		expectErr   bool
		expectedVec []float64
	}{
		"success": {
			response: `{
                "model": "ai/qwen3-embedding",
                "object": "list",
                "usage": { "prompt_tokens": 6, "total_tokens": 6 },
                "data": [
                    {
                        "embedding": [1.1, 2.2, 3.3],
                        "index": 0,
                        "object": "embedding"
                    }
                ]
            }`,
			statusCode:  http.StatusOK,
			model:       "ai/qwen3-embedding",
			input:       "A dog is an animal",
			expectedVec: []float64{1.1, 2.2, 3.3},
		},
		"no-embedding-data": {
			response: `{
                "model": "ai/qwen3-embedding",
                "object": "list",
                "usage": { "prompt_tokens": 6, "total_tokens": 6 },
                "data": []
            }`,
			statusCode: http.StatusOK,
			model:      "ai/qwen3-embedding",
			input:      "A dog is an animal",
			expectErr:  true,
		},
		"server-error": {
			response:   `Internal Server Error`,
			statusCode: http.StatusInternalServerError,
			model:      "ai/qwen3-embedding",
			input:      "A dog is an animal",
			expectErr:  true,
		},
		"invalid-json": {
			response:   `{invalid json}`,
			statusCode: http.StatusOK,
			model:      "ai/qwen3-embedding",
			input:      "A dog is an animal",
			expectErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response)) //nolint:errcheck
			}))
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewLLMClientAdapter(client)

			vec, err := adapter.Embed(context.Background(), tt.model, tt.input)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedVec, vec)
		})
	}
}

func TestInitLLMClient_Initialize(t *testing.T) {
	i := InitLLMClient{}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	r, err := depend.Resolve[domain.LLMClient]()
	assert.NotNil(t, r)
	assert.NoError(t, err)
}
