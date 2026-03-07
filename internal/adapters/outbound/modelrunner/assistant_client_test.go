package modelrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"

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
func collectStreamEvents(ctx context.Context, adapter AssistantClient, req assistant.TurnRequest) ([]assistant.EventType, []string, *assistant.TurnCompleted, error) {
	var eventTypes []assistant.EventType
	var deltaTexts []string
	var doneEvent *assistant.TurnCompleted

	err := adapter.RunTurn(ctx, req, func(_ context.Context, eventType assistant.EventType, data any) error {
		eventTypes = append(eventTypes, eventType)

		switch eventType {
		case assistant.EventType_MessageDelta:
			delta := data.(assistant.MessageDelta)
			deltaTexts = append(deltaTexts, delta.Text)
		case assistant.EventType_TurnCompleted:
			done := data.(assistant.TurnCompleted)
			doneEvent = &done
		}
		return nil
	})

	return eventTypes, deltaTexts, doneEvent, err
}

func TestAssistantClientAdapter_RunTurn(t *testing.T) {
	t.Parallel()

	req := assistant.TurnRequest{
		Stream: true,
		Model:  "test-model",
		Messages: []assistant.Message{
			{Role: "user", Content: "test"},
		},
	}
	tests := map[string]struct {
		req             assistant.TurnRequest
		chunks          []StreamChunk
		expectErr       bool
		expectedEvents  []assistant.EventType
		expectedContent string
	}{
		"multiple-deltas": {
			req: req,
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "Hello"}}}},
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: " "}}}},
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "world"}}}, Usage: &Usage{PromptTokens: 5, CompletionTokens: 5, TotalTokens: 10}},
			},
			expectedEvents: []assistant.EventType{
				assistant.EventType_TurnStarted,
				assistant.EventType_MessageDelta,
				assistant.EventType_MessageDelta,
				assistant.EventType_MessageDelta,
				assistant.EventType_TurnCompleted,
			},
			expectedContent: "Hello world",
		},
		"empty-delta": {
			req: req,
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: ""}}}},
			},
			expectedEvents: []assistant.EventType{
				assistant.EventType_TurnStarted,
				assistant.EventType_TurnCompleted,
			},
			expectedContent: "",
		},
		"with-tool-calls": {
			req: assistant.TurnRequest{
				Model: "test-model",
				Messages: []assistant.Message{
					{
						Role: assistant.ChatRole_Assistant,
						ActionCalls: []assistant.ActionCall{
							{
								ID:    "toolcall-1",
								Name:  "list_todos",
								Input: `{"search_term":"books","page":1,"page_size":5}`,
							},
						},
					},
					{
						Role:         assistant.ChatRole_Tool,
						ActionCallID: common.Ptr("toolcall-1"),
						Content:      `{"todos":[{"id":1,"text":"Buy book","done":false}]}`,
					},
				},
				AvailableActions: []assistant.ActionDefinition{
					{
						Name: "search_web",
						Input: assistant.ActionInput{
							Type: "object",
							Fields: map[string]assistant.ActionField{
								"search_term": {Type: "string", Description: "The search query", Required: true},
							},
						},
					},
				},
			},
			chunks: []StreamChunk{
				{
					Choices: []StreamChunkChoice{
						{
							Delta: StreamChunkDelta{
								ToolCalls: []ToolCallChunk{
									{
										ID: "toolcall-1",
										Function: ToolCallChunkFunction{
											Name: "list_todos", Arguments: `{"search_term":"books","page":1,"page_size":5}`,
										},
									},
								},
							},
						},
					},
				},
			},

			expectedEvents: []assistant.EventType{
				assistant.EventType_TurnStarted,
				assistant.EventType_ActionRequested,
				assistant.EventType_TurnCompleted,
			},
			expectedContent: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := createStreamingServer(tt.chunks)
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewAssistantClientAdapter(client)

			eventTypes, deltaTexts, _, err := collectStreamEvents(t.Context(), adapter, tt.req)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, eventTypes)

			combined := strings.Join(deltaTexts, "")
			assert.Equal(t, tt.expectedContent, combined)

		})
	}
}

func TestAssistantClientAdapter_RunTurn_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewDRMAPIClient(server.URL, "", server.Client())
	adapter := NewAssistantClientAdapter(client)

	req := assistant.TurnRequest{
		Model: "test-model",
		Messages: []assistant.Message{
			{Role: "user", Content: "test"},
		},
	}

	err := adapter.RunTurn(t.Context(), req, func(_ context.Context, eventType assistant.EventType, data any) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestAssistantClientAdapter_RunTurnSync(t *testing.T) {
	t.Parallel()

	temp := 0.5
	topP := 0.9

	tests := map[string]struct {
		response     string
		statusCode   int
		req          assistant.TurnRequest
		expectErr    bool
		expectedResp string
		validateReq  func(*testing.T, *ChatRequest)
	}{
		"success": {
			response:   `{"choices":[{"message":{"role":"assistant","content":"Hello!"}}],"usage": {"completion_tokens": 10,"prompt_tokens": 10,"total_tokens": 20}}`,
			statusCode: http.StatusOK,
			req: assistant.TurnRequest{
				Model: "test-model",
				Messages: []assistant.Message{
					{Role: "user", Content: "hi"},
				},
			},
			expectedResp: "Hello!",
		},
		"with-params": {
			response:   `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`,
			statusCode: http.StatusOK,
			req: assistant.TurnRequest{
				Model:       "test-model",
				Temperature: &temp,
				TopP:        &topP,
				Messages: []assistant.Message{
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
			req: assistant.TurnRequest{
				Model: "test-model",
				Messages: []assistant.Message{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
		"server-error": {
			response:   `Internal Server Error`,
			statusCode: http.StatusInternalServerError,
			req: assistant.TurnRequest{
				Model: "test-model",
				Messages: []assistant.Message{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
		"invalid-json": {
			response:   `{invalid json}`,
			statusCode: http.StatusOK,
			req: assistant.TurnRequest{
				Model: "test-model",
				Messages: []assistant.Message{
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
			adapter := NewAssistantClientAdapter(client)

			resp, err := adapter.RunTurnSync(t.Context(), tt.req)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResp, resp.Content)

			if tt.validateReq != nil && capturedReq != nil {
				tt.validateReq(t, capturedReq)
			}
		})
	}
}

func TestAssistantClientAdapter_RunTurnSync_ValidationErrors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`)) //nolint:errcheck
	}))
	defer server.Close()

	client := NewDRMAPIClient(server.URL, "", server.Client())
	adapter := NewAssistantClientAdapter(client)

	tests := map[string]struct {
		req assistant.TurnRequest
	}{
		"no-model":    {req: assistant.TurnRequest{Messages: []assistant.Message{{Role: "user", Content: "hi"}}}},
		"no-messages": {req: assistant.TurnRequest{Model: "test"}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := adapter.RunTurnSync(t.Context(), tt.req)
			assert.Error(t, err)
		})
	}
}

func TestAssistantClientAdapter_ListAvailableModels(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		response   string
		statusCode int
		expectErr  bool
		expected   []assistant.ModelInfo
	}{
		"success": {
			statusCode: http.StatusOK,
			response: `{
                "object": "list",
                "data": [
                    { "id": "docker.io/ai/qwen3-embedding" },
                    { "id": "docker.io/ai/llama3" }
                ]
            }`,
			expected: []assistant.ModelInfo{
				{ID: "docker.io/ai/qwen3-embedding", Name: "qwen3-embedding", Kind: assistant.ModelKindEmbedding},
				{ID: "docker.io/ai/llama3", Name: "llama3", Kind: assistant.ModelKindAssistant},
			},
		},
		"empty-list": {
			statusCode: http.StatusOK,
			response: `{
                "object": "list",
                "data": []
            }`,
			expected: []assistant.ModelInfo{},
		},
		"server-error": {
			statusCode: http.StatusInternalServerError,
			response:   "Internal Server Error",
			expectErr:  true,
		},
		"invalid-json": {
			statusCode: http.StatusOK,
			response:   `{invalid json}`,
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
			adapter := NewAssistantClientAdapter(client)

			models, err := adapter.ListAvailableModels(t.Context())

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, models)
		})
	}
}

func TestAssistantClientAdapter_ListModels(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		response   string
		statusCode int
		expectErr  bool
		expected   []assistant.ModelCapabilities
	}{
		"success-filters-embeddings": {
			statusCode: http.StatusOK,
			response: `{
                "object": "list",
                "data": [
                    { "id": "docker.io/ai/qwen3-embedding" },
                    { "id": "docker.io/ai/llama3" },
                    { "id": "gpt-4o-mini" }
                ]
            }`,
			expected: []assistant.ModelCapabilities{
				{
					ID:                "docker.io/ai/llama3",
					Name:              "llama3",
					SupportsStreaming: true,
					SupportsActions:   true,
				},
				{
					ID:                "gpt-4o-mini",
					Name:              "gpt-4o-mini",
					SupportsStreaming: true,
					SupportsActions:   true,
				},
			},
		},
		"empty-list": {
			statusCode: http.StatusOK,
			response: `{
                "object": "list",
                "data": []
            }`,
			expected: []assistant.ModelCapabilities{},
		},
		"only-embeddings": {
			statusCode: http.StatusOK,
			response: `{
                "object": "list",
                "data": [
                    { "id": "text-embed-3-small" },
                    { "id": "docker.io/ai/qwen3-embedding" }
                ]
            }`,
			expected: []assistant.ModelCapabilities{},
		},
		"server-error": {
			statusCode: http.StatusInternalServerError,
			response:   "Internal Server Error",
			expectErr:  true,
		},
		"invalid-json": {
			statusCode: http.StatusOK,
			response:   `{invalid json}`,
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
			adapter := NewAssistantClientAdapter(client)

			models, err := adapter.ListModels(t.Context())

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, models)
		})
	}
}
