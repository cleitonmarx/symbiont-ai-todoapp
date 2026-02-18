package modelrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
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
func collectStreamEvents(adapter AssistantClient, req domain.AssistantTurnRequest) ([]domain.AssistantEventType, []string, *domain.AssistantTurnCompleted, error) {
	var eventTypes []domain.AssistantEventType
	var deltaTexts []string
	var doneEvent *domain.AssistantTurnCompleted

	err := adapter.RunTurn(context.Background(), req, func(eventType domain.AssistantEventType, data any) error {
		eventTypes = append(eventTypes, eventType)

		switch eventType {
		case domain.AssistantEventType_MessageDelta:
			delta := data.(domain.AssistantMessageDelta)
			deltaTexts = append(deltaTexts, delta.Text)
		case domain.AssistantEventType_TurnCompleted:
			done := data.(domain.AssistantTurnCompleted)
			doneEvent = &done
		}
		return nil
	})

	return eventTypes, deltaTexts, doneEvent, err
}

func TestAssistantClientAdapter_RunTurn(t *testing.T) {
	req := domain.AssistantTurnRequest{
		Stream: true,
		Model:  "test-model",
		Messages: []domain.AssistantMessage{
			{Role: "user", Content: "test"},
		},
	}
	tests := map[string]struct {
		req             domain.AssistantTurnRequest
		chunks          []StreamChunk
		expectErr       bool
		expectedEvents  []domain.AssistantEventType
		expectedContent string
	}{
		"multiple-deltas": {
			req: req,
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "Hello"}}}},
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: " "}}}},
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "world"}}}, Usage: &Usage{PromptTokens: 5, CompletionTokens: 5, TotalTokens: 10}},
			},
			expectedEvents: []domain.AssistantEventType{
				domain.AssistantEventType_TurnStarted,
				domain.AssistantEventType_MessageDelta,
				domain.AssistantEventType_MessageDelta,
				domain.AssistantEventType_MessageDelta,
				domain.AssistantEventType_TurnCompleted,
			},
			expectedContent: "Hello world",
		},
		"empty-delta": {
			req: req,
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: ""}}}},
			},
			expectedEvents: []domain.AssistantEventType{
				domain.AssistantEventType_TurnStarted,
				domain.AssistantEventType_TurnCompleted,
			},
			expectedContent: "",
		},
		"with-tool-calls": {
			req: domain.AssistantTurnRequest{
				Model: "test-model",
				Messages: []domain.AssistantMessage{
					{
						Role: domain.ChatRole_Assistant,
						ActionCalls: []domain.AssistantActionCall{
							{
								ID:    "toolcall-1",
								Name:  "list_todos",
								Input: `{"search_term":"books","page":1,"page_size":5}`,
							},
						},
					},
					{
						Role:         domain.ChatRole_Tool,
						ActionCallID: common.Ptr("toolcall-1"),
						Content:      `{"todos":[{"id":1,"text":"Buy book","done":false}]}`,
					},
				},
				AvailableActions: []domain.AssistantActionDefinition{
					{
						Name: "search_web",
						Input: domain.AssistantActionInput{
							Type: "object",
							Fields: map[string]domain.AssistantActionField{
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

			expectedEvents: []domain.AssistantEventType{
				domain.AssistantEventType_TurnStarted,
				domain.AssistantEventType_ActionRequested,
				domain.AssistantEventType_TurnCompleted,
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

			eventTypes, deltaTexts, _, err := collectStreamEvents(adapter, tt.req)

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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewDRMAPIClient(server.URL, "", server.Client())
	adapter := NewAssistantClientAdapter(client)

	req := domain.AssistantTurnRequest{
		Model: "test-model",
		Messages: []domain.AssistantMessage{
			{Role: "user", Content: "test"},
		},
	}

	err := adapter.RunTurn(context.Background(), req, func(eventType domain.AssistantEventType, data interface{}) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestAssistantClientAdapter_RunTurnSync(t *testing.T) {
	temp := 0.5
	topP := 0.9

	tests := map[string]struct {
		response     string
		statusCode   int
		req          domain.AssistantTurnRequest
		expectErr    bool
		expectedResp string
		validateReq  func(*testing.T, *ChatRequest)
	}{
		"success": {
			response:   `{"choices":[{"message":{"role":"assistant","content":"Hello!"}}],"usage": {"completion_tokens": 10,"prompt_tokens": 10,"total_tokens": 20}}`,
			statusCode: http.StatusOK,
			req: domain.AssistantTurnRequest{
				Model: "test-model",
				Messages: []domain.AssistantMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectedResp: "Hello!",
		},
		"with-params": {
			response:   `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`,
			statusCode: http.StatusOK,
			req: domain.AssistantTurnRequest{
				Model:       "test-model",
				Temperature: &temp,
				TopP:        &topP,
				Messages: []domain.AssistantMessage{
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
			req: domain.AssistantTurnRequest{
				Model: "test-model",
				Messages: []domain.AssistantMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
		"server-error": {
			response:   `Internal Server Error`,
			statusCode: http.StatusInternalServerError,
			req: domain.AssistantTurnRequest{
				Model: "test-model",
				Messages: []domain.AssistantMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
		"invalid-json": {
			response:   `{invalid json}`,
			statusCode: http.StatusOK,
			req: domain.AssistantTurnRequest{
				Model: "test-model",
				Messages: []domain.AssistantMessage{
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

			resp, err := adapter.RunTurnSync(context.Background(), tt.req)

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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`)) //nolint:errcheck
	}))
	defer server.Close()

	client := NewDRMAPIClient(server.URL, "", server.Client())
	adapter := NewAssistantClientAdapter(client)

	tests := map[string]struct {
		req domain.AssistantTurnRequest
	}{
		"no-model":    {req: domain.AssistantTurnRequest{Messages: []domain.AssistantMessage{{Role: "user", Content: "hi"}}}},
		"no-messages": {req: domain.AssistantTurnRequest{Model: "test"}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := adapter.RunTurnSync(context.Background(), tt.req)
			assert.Error(t, err)
		})
	}
}

func TestAssistantClientAdapter_VectorizeTodo(t *testing.T) {
	todo := domain.Todo{Title: "Test", DueDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Status: domain.TodoStatus_OPEN}
	tests := map[string]struct {
		response           string
		statusCode         int
		model              string
		expectRequestInput string
		expectErr          bool
		expectedVec        []float64
	}{
		"success-with-embeddinggemma": {
			response: `{
                "model": "ai/embeddinggemma",
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
			statusCode:         http.StatusOK,
			model:              "ai/embeddinggemma",
			expectRequestInput: "title: none | text: Test",
			expectedVec:        []float64{1.1, 2.2, 3.3},
		},
		"success-with-default-embedding-generator": {
			response: `{
                "model": "ai/otherembeddingmodel",
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
			statusCode:         http.StatusOK,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "title:'Test'\ndue_date:'2024-01-01T00:00:00Z'\nstatus:'OPEN'",
			expectedVec:        []float64{1.1, 2.2, 3.3},
		},
		"no-embedding-data": {
			response: `{
                "model": "ai/otherembeddingmodel",
                "object": "list",
                "usage": { "prompt_tokens": 6, "total_tokens": 6 },
                "data": []
            }`,
			statusCode:         http.StatusOK,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "title:'Test'\ndue_date:'2024-01-01T00:00:00Z'\nstatus:'OPEN'",
			expectErr:          true,
		},
		"server-error": {
			response:           `Internal Server Error`,
			statusCode:         http.StatusInternalServerError,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "title:'Test'\ndue_date:'2024-01-01T00:00:00Z'\nstatus:'OPEN'",
			expectErr:          true,
		},
		"invalid-json": {
			response:           `{invalid json}`,
			statusCode:         http.StatusOK,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "title:'Test'\ndue_date:'2024-01-01T00:00:00Z'\nstatus:'OPEN'",
			expectErr:          true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req EmbeddingsRequest
				json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
				assert.Equal(t, tt.expectRequestInput, req.Input)

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response)) //nolint:errcheck
			}))
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewAssistantClientAdapter(client)

			vec, err := adapter.VectorizeTodo(context.Background(), tt.model, todo)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedVec, vec.Vector)
		})
	}
}

func TestAssistantClientAdapter_VectorizeQuery(t *testing.T) {
	searchInput := "Find todos about books"
	tests := map[string]struct {
		response           string
		statusCode         int
		model              string
		expectRequestInput string
		expectErr          bool
		expectedVec        []float64
	}{
		"success-with-gemma-embedding": {
			response: `{
				"model": "ai/embeddinggemma",
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
			statusCode:         http.StatusOK,
			model:              "ai/embeddinggemma",
			expectRequestInput: "task: search result | query: Find todos about books",
			expectedVec:        []float64{1.1, 2.2, 3.3},
		},
		"success-with-default-embedding-generator": {
			response: `{
				"model": "ai/otherembeddingmodel",
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
			statusCode:         http.StatusOK,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "Find todos about books",
			expectedVec:        []float64{1.1, 2.2, 3.3},
		},
		"no-embedding-data": {
			response: `{
				"model": "ai/otherembeddingmodel",
				"object": "list",
				"usage": { "prompt_tokens": 6, "total_tokens": 6 },
				"data": []
			}`,
			statusCode:         http.StatusOK,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "Find todos about books",
			expectErr:          true,
		},
		"server-error": {
			response:           `Internal Server Error`,
			statusCode:         http.StatusInternalServerError,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "Find todos about books",
			expectErr:          true,
		},
		"invalid-json": {
			response:           `{invalid json}`,
			statusCode:         http.StatusOK,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "Find todos about books",
			expectErr:          true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req EmbeddingsRequest
				json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
				assert.Equal(t, tt.expectRequestInput, req.Input)

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response)) //nolint:errcheck
			}))
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewAssistantClientAdapter(client)

			vec, err := adapter.VectorizeQuery(context.Background(), tt.model, searchInput)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedVec, vec.Vector)
		})
	}
}

func TestAssistantClientAdapter_ListAvailableModels(t *testing.T) {
	tests := map[string]struct {
		response   string
		statusCode int
		expectErr  bool
		expected   []domain.ModelInfo
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
			expected: []domain.ModelInfo{
				{Name: "qwen3-embedding", Kind: domain.ModelKindEmbedding},
				{Name: "llama3", Kind: domain.ModelKindAssistant},
			},
		},
		"empty-list": {
			statusCode: http.StatusOK,
			response: `{
                "object": "list",
                "data": []
            }`,
			expected: []domain.ModelInfo{},
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

			models, err := adapter.ListAvailableModels(context.Background())

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, models)
		})
	}
}

func TestInitAssistantClient_Initialize(t *testing.T) {
	i := InitAssistantClient{}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	r, err := depend.Resolve[domain.Assistant]()
	assert.NotNil(t, r)
	assert.NoError(t, err)
}
