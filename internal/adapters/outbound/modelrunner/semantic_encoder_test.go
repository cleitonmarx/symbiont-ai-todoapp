package modelrunner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/stretchr/testify/assert"
)

func TestSemanticEncoder_VectorizeTodo(t *testing.T) {
	t.Parallel()

	todo := todo.Todo{Title: "Test", DueDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Status: todo.Status_OPEN}
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
			expectRequestInput: "Test",
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
			expectRequestInput: "Test",
			expectErr:          true,
		},
		"server-error": {
			response:           `Internal Server Error`,
			statusCode:         http.StatusInternalServerError,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "Test",
			expectErr:          true,
		},
		"invalid-json": {
			response:           `{invalid json}`,
			statusCode:         http.StatusOK,
			model:              "ai/otherembeddingmodel",
			expectRequestInput: "Test",
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
			adapter := NewSemanticEncoder(client)

			vec, err := adapter.VectorizeTodo(t.Context(), tt.model, todo)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedVec, vec.Vector)
		})
	}
}

func TestSemanticEncoder_VectorizeQuery(t *testing.T) {
	t.Parallel()

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
			adapter := NewSemanticEncoder(client)

			vec, err := adapter.VectorizeQuery(t.Context(), tt.model, searchInput)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedVec, vec.Vector)
		})
	}
}

func TestSemanticEncoder_VectorizeSkillDefinition(t *testing.T) {
	t.Parallel()

	baseSkill := assistant.SkillDefinition{
		Name:      "todo-mutation-safety",
		UseWhen:   "update todos",
		AvoidWhen: "chat only",
		Tags:      []string{"todos", "mutation"},
		Tools:     []string{"fetch_todos", "update_todos"},
		Content:   "Never invent IDs",
	}

	tests := map[string]struct {
		model         string
		skill         assistant.SkillDefinition
		expectInputs  []string
		expectErr     bool
		expectedUse   []float64
		expectedAvoid []float64
	}{
		"default-model-with-avoid-embedding": {
			model: "ai/otherembeddingmodel",
			skill: baseSkill,
			expectInputs: []string{
				"todo-mutation-safety\nupdate todos\nRelated terms: todos, mutation\nActions/tools: fetch_todos, update_todos",
				"todo-mutation-safety\nAvoid when: chat only",
			},
			expectedUse:   []float64{1.1, 2.2, 3.3},
			expectedAvoid: []float64{4.4, 5.5, 6.6},
		},
		"default-model-with-content-line-property": {
			model: "ai/otherembeddingmodel",
			skill: assistant.SkillDefinition{
				Name:                  "todo-delete",
				UseWhen:               "delete todos",
				AvoidWhen:             "chat only",
				Tags:                  []string{"todos", "delete"},
				Tools:                 []string{"fetch_todos", "delete_todos"},
				EmbedFirstContentLine: true,
				Content:               "Goal: execute deletions safely and only on confirmed targets.",
			},
			expectInputs: []string{
				"todo-delete\ndelete todos\nGoal: execute deletions safely and only on confirmed targets.\nRelated terms: todos, delete\nActions/tools: fetch_todos, delete_todos",
				"todo-delete\nAvoid when: chat only",
			},
			expectedUse:   []float64{1.1, 2.2, 3.3},
			expectedAvoid: []float64{4.4, 5.5, 6.6},
		},
		"gemma-model-without-avoid-embedding": {
			model: "ai/embeddinggemma",
			skill: assistant.SkillDefinition{
				Name:    "todo-fetch",
				UseWhen: "list todos",
				Tools:   []string{"fetch_todos"},
				Content: "Use fetch_todos for read operations",
			},
			expectInputs: []string{
				"title: todo-fetch | text: list todos\nActions/tools: fetch_todos",
			},
			expectedUse:   []float64{1.1, 2.2, 3.3},
			expectedAvoid: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			callCount := 0
			gotInputs := make([]string, 0, 2)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req EmbeddingsRequest
				json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
				input, _ := req.Input.(string)
				gotInputs = append(gotInputs, input)

				response := `{
					"model": "ai/embeddingmodel",
					"object": "list",
					"usage": { "prompt_tokens": 6, "total_tokens": 6 },
					"data": [{ "embedding": [1.1, 2.2, 3.3], "index": 0, "object": "embedding" }]
				}`
				if callCount > 0 {
					response = `{
						"model": "ai/embeddingmodel",
						"object": "list",
						"usage": { "prompt_tokens": 6, "total_tokens": 6 },
						"data": [{ "embedding": [4.4, 5.5, 6.6], "index": 0, "object": "embedding" }]
					}`
				}
				callCount++

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response)) //nolint:errcheck
			}))
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewSemanticEncoder(client)

			useVec, avoidVec, err := adapter.VectorizeSkillDefinition(t.Context(), tt.model, tt.skill)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectInputs, gotInputs)
			assert.Equal(t, tt.expectedUse, useVec.Vector)
			assert.Equal(t, tt.expectedAvoid, avoidVec.Vector)
		})
	}
}
