package mcp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPRegistry_ListEmbeddings_MapsToolSchema(t *testing.T) {
	session := &fakeSession{
		listResults: []*mcp.ListToolsResult{
			{
				Tools: []*mcp.Tool{
					{
						Name:        "create_task",
						Description: "Creates one task",
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"title": map[string]any{
									"type":        "string",
									"description": "Task title",
								},
								"priority": map[string]any{
									"type": []any{"integer", "null"},
								},
							},
							"required": []any{"title"},
						},
					},
				},
			},
		},
	}

	registry := newMCPRegistryWithConnector(
		Config{
			Endpoint: "http://localhost:8811/mcp",
		},
		&fakeConnector{session: session},
		nil,
		"",
	)
	require.NoError(t, registry.initializeActions(t.Context()))

	embeddingsList := registry.ListEmbeddings(t.Context())
	require.Len(t, embeddingsList, 1)

	def := embeddingsList[0].Action.Definition()
	assert.Equal(t, "create_task", def.Name)
	assert.Equal(t, "Creates one task", def.Description)
	assert.Equal(t, "object", def.Input.Type)
	assert.Equal(t, "string", def.Input.Fields["title"].Type)
	assert.True(t, def.Input.Fields["title"].Required)
	assert.Equal(t, "Task title", def.Input.Fields["title"].Description)
	assert.Equal(t, "integer|null", def.Input.Fields["priority"].Type)
}

func TestMCPRegistry_Execute_CallsTool(t *testing.T) {
	session := &fakeSession{
		listResults: []*mcp.ListToolsResult{
			{
				Tools: []*mcp.Tool{
					{
						Name:        "fetch",
						Description: "Fetches content",
						InputSchema: map[string]any{"type": "object"},
					},
				},
			},
		},
		callResult: &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "done"},
			},
		},
	}

	registry := newMCPRegistryWithConnector(
		Config{
			Endpoint: "http://localhost:8811/mcp",
		},
		&fakeConnector{session: session},
		nil,
		"",
	)
	require.NoError(t, registry.initializeActions(t.Context()))

	call := domain.AssistantActionCall{
		ID:    "call-1",
		Name:  "fetch",
		Input: `{"url":"https://example.com"}`,
	}
	msg := registry.Execute(context.Background(), call, nil)

	require.NotNil(t, msg.ActionCallID)
	assert.Equal(t, "call-1", *msg.ActionCallID)
	assert.Equal(t, domain.ChatRole_Tool, msg.Role)
	assert.Equal(t, "done", msg.Content)

	require.NotNil(t, session.lastCallParams)
	assert.Equal(t, "fetch", session.lastCallParams.Name)
	assert.Equal(t, "https://example.com", session.lastCallParams.Arguments.(map[string]any)["url"])
}

func TestRegistry_Execute_InvalidArguments(t *testing.T) {
	session := &fakeSession{
		listResults: []*mcp.ListToolsResult{
			{
				Tools: []*mcp.Tool{
					{
						Name:        "fetch",
						Description: "Fetches content",
						InputSchema: map[string]any{"type": "object"},
					},
				},
			},
		},
	}

	registry := newMCPRegistryWithConnector(
		Config{Endpoint: "http://localhost:8811/mcp"},
		&fakeConnector{session: session},
		nil,
		"",
	)
	require.NoError(t, registry.initializeActions(t.Context()))

	msg := registry.Execute(context.Background(), domain.AssistantActionCall{
		ID:    "call-2",
		Name:  "fetch",
		Input: `[]`,
	}, nil)

	assert.Contains(t, msg.Content, "invalid_arguments")
}

func TestRegistry_Execute_IsErrorPrefix(t *testing.T) {
	session := &fakeSession{
		listResults: []*mcp.ListToolsResult{
			{
				Tools: []*mcp.Tool{
					{
						Name:        "fetch",
						Description: "Fetches content",
						InputSchema: map[string]any{"type": "object"},
					},
				},
			},
		},
		callResult: &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: "failed"},
			},
		},
	}

	registry := newMCPRegistryWithConnector(
		Config{Endpoint: "http://localhost:8811/mcp"},
		&fakeConnector{session: session},
		nil,
		"",
	)
	require.NoError(t, registry.initializeActions(t.Context()))

	msg := registry.Execute(context.Background(), domain.AssistantActionCall{
		ID:    "call-3",
		Name:  "fetch",
		Input: `{}`,
	}, nil)

	assert.Equal(t, "error: failed", msg.Content)
}

type fakeConnector struct {
	session mcpSession
	err     error
}

func (c *fakeConnector) Connect(context.Context) (mcpSession, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.session, nil
}

type fakeSession struct {
	listResults    []*mcp.ListToolsResult
	listErr        error
	callResult     *mcp.CallToolResult
	callErr        error
	lastCallParams *mcp.CallToolParams
	listCalls      int
}

func (s *fakeSession) ListTools(_ context.Context, _ *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if len(s.listResults) == 0 {
		return &mcp.ListToolsResult{}, nil
	}
	index := min(s.listCalls, len(s.listResults)-1)
	s.listCalls++
	return s.listResults[index], nil
}

func (s *fakeSession) CallTool(_ context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	s.lastCallParams = params
	if s.callErr != nil {
		return nil, s.callErr
	}
	return s.callResult, nil
}

func (s *fakeSession) Close() error { return nil }

func TestRegistry_Execute_UnknownAction(t *testing.T) {
	registry := newMCPRegistryWithConnector(
		Config{Endpoint: "http://localhost:8811/mcp"},
		&fakeConnector{session: &fakeSession{}},
		nil,
		"",
	)

	msg := registry.Execute(context.Background(), domain.AssistantActionCall{
		ID:    "call-unknown",
		Name:  "missing_tool",
		Input: `{}`,
	}, nil)

	assert.Contains(t, msg.Content, "unknown_action")
}

func TestRegistry_InitializeActions_AppliesToolOverrides(t *testing.T) {
	session := &fakeSession{
		listResults: []*mcp.ListToolsResult{
			{
				Tools: []*mcp.Tool{
					{
						Name:        "search",
						Description: "Original description",
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"query": map[string]any{
									"type":        "string",
									"description": "Query",
								},
							},
							"required": []any{"query"},
						},
					},
				},
			},
		},
	}

	registry := newMCPRegistryWithConnector(
		Config{
			Endpoint: "http://localhost:8811/mcp",
		},
		&fakeConnector{session: session},
		nil,
		"",
	)

	require.NoError(t, registry.initializeActions(context.Background()))
	defs := registry.ListEmbeddings(context.Background())
	require.Len(t, defs, 1)

	def := defs[0].Action.Definition()
	assert.Equal(t, "search", def.Name)
	assert.Equal(t, "Search the web with DuckDuckGo and return concise result snippets with source links.", def.Description)
	assert.Equal(t, "object", def.Input.Type)
	assert.Equal(t, "string", def.Input.Fields["query"].Type)
	assert.Equal(t, "Search query in natural language.", def.Input.Fields["query"].Description)
	assert.True(t, def.Input.Fields["query"].Required)
	assert.Equal(t, "integer", def.Input.Fields["max_results"].Type)
	assert.Equal(t, "Use to gather external information or references before deciding or answering.", def.Hints.UseWhen)
	assert.Equal(t, "Do not use for todo CRUD operations or when the user request is fully internal to the app.", def.Hints.AvoidWhen)
	assert.Equal(t, "Always send a specific query and include max_results. Default max_results=2. Never exceed 3 unless the user explicitly asks for broad research. Prefer one focused query per turn.", def.Hints.ArgRules)
}

func TestParseActionCallArguments_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    map[string]any
		wantErr string
	}{
		{
			name:  "empty-input",
			input: "",
			want:  map[string]any{},
		},
		{
			name:  "valid-object",
			input: `{"query":"hello","max_results":2}`,
			want: map[string]any{
				"query":       "hello",
				"max_results": float64(2),
			},
		},
		{
			name:    "invalid-json",
			input:   `{"query":`,
			wantErr: "unexpected EOF",
		},
		{
			name:    "non-object-json",
			input:   `["a"]`,
			wantErr: "action arguments must be a JSON object",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseActionCallArguments(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSchemaToInput_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema any
		assert func(*testing.T, domain.AssistantActionInput)
	}{
		{
			name:   "nil-schema",
			schema: nil,
			assert: func(t *testing.T, got domain.AssistantActionInput) {
				assert.Equal(t, "object", got.Type)
				assert.Empty(t, got.Fields)
			},
		},
		{
			name: "simple-properties-required",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "Task title",
					},
				},
				"required": []any{"title"},
			},
			assert: func(t *testing.T, got domain.AssistantActionInput) {
				assert.Equal(t, "object", got.Type)
				require.Contains(t, got.Fields, "title")
				assert.Equal(t, "string", got.Fields["title"].Type)
				assert.Equal(t, "Task title", got.Fields["title"].Description)
				assert.True(t, got.Fields["title"].Required)
			},
		},
		{
			name: "composed-field-type",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"priority": map[string]any{
						"oneOf": []any{
							map[string]any{"type": "integer"},
							map[string]any{"type": "null"},
						},
					},
				},
			},
			assert: func(t *testing.T, got domain.AssistantActionInput) {
				require.Contains(t, got.Fields, "priority")
				assert.Equal(t, "integer|null", got.Fields["priority"].Type)
				assert.False(t, got.Fields["priority"].Required)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, schemaToInput(tt.schema))
		})
	}
}

func TestParseToolOverrideDefinitions_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		assert  func(*testing.T, map[string]domain.AssistantActionDefinition, error)
	}{
		{
			name: "valid-yaml",
			content: `
tools:
  - name: search
    description: Search docs
    input:
      type: object
      fields:
        query:
          type: string
          description: Query
          required: true
    hints:
      use_when: When user asks
      avoid_when: Never for writes
      arg_rules: query is required
  - name: "   "
    description: ignored
`,
			assert: func(t *testing.T, got map[string]domain.AssistantActionDefinition, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				require.Contains(t, got, "search")
				assert.Equal(t, "Search docs", got["search"].Description)
				assert.Equal(t, "string", got["search"].Input.Fields["query"].Type)
				assert.Equal(t, "When user asks", got["search"].Hints.UseWhen)
			},
		},
		{
			name:    "invalid-yaml",
			content: "tools: [",
			assert: func(t *testing.T, _ map[string]domain.AssistantActionDefinition, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseToolOverrideDefinitions([]byte(tt.content))
			tt.assert(t, got, err)
		})
	}
}

func TestMergeAssistantActionDefinition_Table(t *testing.T) {
	t.Parallel()

	base := domain.AssistantActionDefinition{
		Name:        "search",
		Description: "base description",
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"query": {Type: "string", Description: "q", Required: true},
			},
		},
		Hints: domain.AssistantActionHints{
			UseWhen:   "base use",
			AvoidWhen: "base avoid",
			ArgRules:  "base args",
		},
	}

	tests := []struct {
		name     string
		override domain.AssistantActionDefinition
		assert   func(*testing.T, domain.AssistantActionDefinition)
	}{
		{
			name: "merge-input-and-keep-hints",
			override: domain.AssistantActionDefinition{
				Description: "override description",
				Input: domain.AssistantActionInput{
					Fields: map[string]domain.AssistantActionField{
						"max_results": {Type: "integer", Description: "max", Required: false},
					},
				},
			},
			assert: func(t *testing.T, got domain.AssistantActionDefinition) {
				assert.Equal(t, "override description", got.Description)
				assert.Equal(t, "base use", got.Hints.UseWhen)
				assert.Contains(t, got.Input.Fields, "query")
				assert.Contains(t, got.Input.Fields, "max_results")
			},
		},
		{
			name: "replace-hints-when-provided",
			override: domain.AssistantActionDefinition{
				Hints: domain.AssistantActionHints{
					UseWhen: "new use",
				},
			},
			assert: func(t *testing.T, got domain.AssistantActionDefinition) {
				assert.Equal(t, "new use", got.Hints.UseWhen)
				assert.Empty(t, got.Hints.AvoidWhen)
				assert.Empty(t, got.Hints.ArgRules)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mergeAssistantActionDefinition(base, tt.override)
			tt.assert(t, got)
		})
	}
}

func TestRenderCallToolResult_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result *mcp.CallToolResult
		assert func(*testing.T, string)
	}{
		{
			name:   "nil-result",
			result: nil,
			assert: func(t *testing.T, got string) {
				assert.Equal(t, "", got)
			},
		},
		{
			name: "structured-content",
			result: &mcp.CallToolResult{
				StructuredContent: map[string]any{"k": "v"},
			},
			assert: func(t *testing.T, got string) {
				assert.Contains(t, got, "k")
				assert.Contains(t, got, "v")
			},
		},
		{
			name: "text-content-joins-non-empty-lines",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "line1"},
					&mcp.TextContent{Text: "   "},
					&mcp.TextContent{Text: "line2"},
				},
			},
			assert: func(t *testing.T, got string) {
				assert.Equal(t, "line1\nline2", got)
			},
		},
		{
			name: "resource-link-content",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.ResourceLink{URI: "https://example.com", Name: "example"},
				},
			},
			assert: func(t *testing.T, got string) {
				assert.Contains(t, got, "resource_link")
				assert.Contains(t, got, "https://example.com")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, renderCallToolResult(tt.result))
		})
	}
}

func TestWithAPIKey_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		headerName string
		apiKey     string
		wantHeader string
		wantValue  string
	}{
		{
			name:       "injects-header",
			headerName: "Authorization",
			apiKey:     "Bearer test-token",
			wantHeader: "Authorization",
			wantValue:  "Bearer test-token",
		},
		{
			name:       "no-key-no-header",
			headerName: "Authorization",
			apiKey:     "",
			wantHeader: "Authorization",
			wantValue:  "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotHeaderVal string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotHeaderVal = r.Header.Get(tt.wantHeader)
				_, _ = io.WriteString(w, "ok")
			}))
			defer server.Close()

			client := withAPIKey(nil, tt.headerName, tt.apiKey)
			resp, err := client.Get(server.URL)
			require.NoError(t, err)
			defer resp.Body.Close() //nolint:errcheck

			assert.Equal(t, tt.wantValue, gotHeaderVal)
		})
	}
}

func TestListAllTools_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		listResult []*mcp.ListToolsResult
		assert     func(*testing.T, []*mcp.Tool, error)
	}{
		{
			name: "nil-response-returns-empty",
			listResult: []*mcp.ListToolsResult{
				nil,
			},
			assert: func(t *testing.T, got []*mcp.Tool, err error) {
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		{
			name: "multi-page-results",
			listResult: []*mcp.ListToolsResult{
				{
					Tools:      []*mcp.Tool{{Name: "search"}},
					NextCursor: "cursor-1",
				},
				{
					Tools:      []*mcp.Tool{{Name: "fetch_content"}},
					NextCursor: "",
				},
			},
			assert: func(t *testing.T, got []*mcp.Tool, err error) {
				require.NoError(t, err)
				require.Len(t, got, 2)
				assert.Equal(t, "search", got[0].Name)
				assert.Equal(t, "fetch_content", got[1].Name)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			session := &fakeSession{listResults: tt.listResult}
			got, err := listAllTools(context.Background(), session)
			tt.assert(t, got, err)
		})
	}
}

func TestAnyToMap_And_AsString_Table(t *testing.T) {
	t.Parallel()

	t.Run("anyToMap", func(t *testing.T) {
		t.Parallel()

		type sample struct {
			A int    `json:"a"`
			B string `json:"b"`
		}

		tests := []struct {
			name    string
			input   any
			wantOK  bool
			wantMap map[string]any
		}{
			{
				name:    "map-input",
				input:   map[string]any{"k": "v"},
				wantOK:  true,
				wantMap: map[string]any{"k": "v"},
			},
			{
				name:    "struct-input",
				input:   sample{A: 1, B: "x"},
				wantOK:  true,
				wantMap: map[string]any{"a": float64(1), "b": "x"},
			},
			{
				name:   "nil-input",
				input:  nil,
				wantOK: false,
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := anyToMap(tt.input)
				assert.Equal(t, tt.wantOK, ok)
				if tt.wantOK {
					assert.Equal(t, tt.wantMap, got)
				}
			})
		}
	})

	t.Run("asString", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			input any
			want  string
		}{
			{name: "nil", input: nil, want: ""},
			{name: "string", input: "abc", want: "abc"},
			{name: "int", input: 42, want: "42"},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.want, asString(tt.input))
			})
		}
	})
}
