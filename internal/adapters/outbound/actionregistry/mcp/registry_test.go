package mcp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPRegistry_Execute_CallsTool(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	registry := newMCPRegistryWithConnector(
		Config{Endpoint: "http://localhost:8811/mcp"},
		&fakeConnector{session: &fakeSession{}},
	)

	msg := registry.Execute(context.Background(), domain.AssistantActionCall{
		ID:    "call-unknown",
		Name:  "missing_tool",
		Input: `{}`,
	}, nil)

	assert.Contains(t, msg.Content, "unknown_action")
}

func TestRegistry_InitializeActions_AppliesToolOverrides(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tool   *mcp.Tool
		assert func(*testing.T, *MCPRegistry)
	}{
		"search": {
			tool: &mcp.Tool{
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
			assert: func(t *testing.T, registry *MCPRegistry) {
				def, found := registry.GetDefinition("search")
				require.True(t, found)
				assert.Equal(t, "search", def.Name)
				assert.Equal(t, "Search the web with DuckDuckGo and return concise result snippets with source links.", def.Description)
				assert.Equal(t, "object", def.Input.Type)
				assert.Equal(t, "string", def.Input.Fields["query"].Type)
				assert.Equal(t, "Search query in natural language.", def.Input.Fields["query"].Description)
				assert.True(t, def.Input.Fields["query"].Required)
				assert.Equal(t, "integer", def.Input.Fields["max_results"].Type)
				assert.Equal(t, "🔎 Searching on the web...", registry.StatusMessage("search"))
			},
		},
		"execute-code": {
			tool: &mcp.Tool{
				Name:        "execute_code",
				Description: "Original execute code description",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code": map[string]any{
							"type":        "string",
							"description": "Original code description",
						},
						"session_id": map[string]any{
							"type":        "integer",
							"description": "Original session_id description",
						},
					},
					"required": []any{"code"},
				},
			},
			assert: func(t *testing.T, registry *MCPRegistry) {
				def, found := registry.GetDefinition("execute_code")
				require.True(t, found)
				assert.Equal(t, "execute_code", def.Name)
				assert.Equal(t, "Execute short self-contained Python code for deterministic calculations, grouping, validation, and data shaping.", def.Description)
				assert.Equal(t, "string", def.Input.Fields["code"].Type)
				assert.Equal(t, "Python code to execute. Make it self-contained and inline the data you need directly in the script unless another tool field explicitly carries variables. End by evaluating `result` or another final expression to return the computed value.", def.Input.Fields["code"].Description)
				assert.True(t, def.Input.Fields["code"].Required)
				assert.Equal(t, "integer", def.Input.Fields["session_id"].Type)
				assert.Equal(t, "Optional interpreter session ID. Omit it on the first call. Reuse the returned session_id only when you intentionally want to continue the same Python session in a later call.", def.Input.Fields["session_id"].Description)
				assert.False(t, def.Input.Fields["session_id"].Required)
				assert.Equal(t, "🧮 Running deterministic code...", registry.StatusMessage("execute_code"))
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session := &fakeSession{
				listResults: []*mcp.ListToolsResult{
					{
						Tools: []*mcp.Tool{tt.tool},
					},
				},
			}

			registry := newMCPRegistryWithConnector(
				Config{
					Endpoint: "http://localhost:8811/mcp",
				},
				&fakeConnector{session: session},
			)

			require.NoError(t, registry.initializeActions(context.Background()))
			tt.assert(t, registry)
		})
	}
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
		{
			name: "nested-array-object-with-format-and-enum",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"todos": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id": map[string]any{
									"type":        "string",
									"description": "Todo id",
								},
								"status": map[string]any{
									"type":   "string",
									"enum":   []any{"OPEN", "DONE"},
									"format": "enum",
								},
							},
							"required": []any{"id"},
						},
					},
				},
				"required": []any{"todos"},
			},
			assert: func(t *testing.T, got domain.AssistantActionInput) {
				require.Contains(t, got.Fields, "todos")
				field := got.Fields["todos"]
				assert.Equal(t, "array", field.Type)
				assert.True(t, field.Required)
				require.NotNil(t, field.Items)
				assert.Equal(t, "object", field.Items.Type)
				require.Contains(t, field.Items.Fields, "id")
				assert.True(t, field.Items.Fields["id"].Required)
				require.Contains(t, field.Items.Fields, "status")
				assert.Equal(t, []any{"OPEN", "DONE"}, field.Items.Fields["status"].Enum)
				assert.Equal(t, "enum", field.Items.Fields["status"].Format)
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
    status_message: Searching docs...
    input:
      type: object
      fields:
        query:
          type: string
          description: Query
          required: true
  - name: "   "
    description: ignored
`,
			assert: func(t *testing.T, got map[string]domain.AssistantActionDefinition, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				require.Contains(t, got, "search")
				assert.Equal(t, "Search docs", got["search"].Description)
				assert.Equal(t, "string", got["search"].Input.Fields["query"].Type)
			},
		},
		{
			name: "valid-yaml-with-approvals-override",
			content: `
tools:
  - name: delete_todos
    approvals:
      required: true
      title: Confirm delete
      description: Destructive action.
      preview_fields:
        - todos[].title
        - todos[].id
      timeout: 45s
`,
			assert: func(t *testing.T, got map[string]domain.AssistantActionDefinition, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				require.Contains(t, got, "delete_todos")
				assert.True(t, got["delete_todos"].Approval.Required)
				assert.Equal(t, "Confirm delete", got["delete_todos"].Approval.Title)
				assert.Equal(t, "Destructive action.", got["delete_todos"].Approval.Description)
				assert.Equal(t, []string{"todos[].title", "todos[].id"}, got["delete_todos"].Approval.PreviewFields)
				assert.Equal(t, 45*time.Second, got["delete_todos"].Approval.Timeout)
			},
		},
		{
			name: "valid-yaml-with-nested-input-fields",
			content: `
tools:
  - name: update_todos
    input:
      type: object
      fields:
        todos:
          type: array
          required: true
          items:
            type: object
            fields:
              id:
                type: string
                required: true
                format: uuid
              status:
                type: string
                required: false
                enum: [OPEN, DONE]
`,
			assert: func(t *testing.T, got map[string]domain.AssistantActionDefinition, err error) {
				require.NoError(t, err)
				require.Contains(t, got, "update_todos")
				todosField := got["update_todos"].Input.Fields["todos"]
				assert.Equal(t, "array", todosField.Type)
				assert.True(t, todosField.Required)
				require.NotNil(t, todosField.Items)
				require.Contains(t, todosField.Items.Fields, "id")
				assert.Equal(t, "uuid", todosField.Items.Fields["id"].Format)
				assert.Equal(t, []any{"OPEN", "DONE"}, todosField.Items.Fields["status"].Enum)
			},
		},
		{
			name: "invalid-yaml-with-approval-timeout-format",
			content: `
tools:
  - name: delete_todos
    approvals:
      timeout: 45
`,
			assert: func(t *testing.T, _ map[string]domain.AssistantActionDefinition, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid approval timeout")
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

func TestParseToolOverrideStatusMessages_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		assert  func(*testing.T, map[string]string, error)
	}{
		{
			name: "valid-yaml",
			content: `
tools:
  - name: search
    status_message: Searching docs...
  - name: fetch_content
    status_message: "  "
`,
			assert: func(t *testing.T, got map[string]string, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, "Searching docs...", got["search"])
			},
		},
		{
			name:    "invalid-yaml",
			content: "tools: [",
			assert: func(t *testing.T, _ map[string]string, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseToolOverrideStatusMessages([]byte(tt.content))
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
				assert.Contains(t, got.Input.Fields, "query")
				assert.Contains(t, got.Input.Fields, "max_results")
			},
		},
		{
			name: "replace-approval-when-provided",
			override: domain.AssistantActionDefinition{
				Approval: domain.AssistantActionApproval{
					Required: true,
					Title:    "Approve action",
					PreviewFields: []string{
						"todos[].title",
					},
					Timeout: 30 * time.Second,
				},
			},
			assert: func(t *testing.T, got domain.AssistantActionDefinition) {
				assert.True(t, got.Approval.Required)
				assert.Equal(t, "Approve action", got.Approval.Title)
				assert.Equal(t, []string{"todos[].title"}, got.Approval.PreviewFields)
				assert.Equal(t, 30*time.Second, got.Approval.Timeout)
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
