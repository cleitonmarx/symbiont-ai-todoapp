package mcp

import (
	"context"
	"os"
	"path/filepath"
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
						Name:        "fetch",
						Description: "Original description",
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"url": map[string]any{
									"type":        "string",
									"description": "URL",
								},
							},
							"required": []any{"url"},
						},
					},
				},
			},
		},
	}

	overridePath := filepath.Join(t.TempDir(), "overrides.yaml")
	overrideYAML := `
tools:
  - name: fetch
    description: Fetch web content with constraints
    input:
      type: object
      fields:
        url:
          type: string
          description: Absolute URL
          required: true
        max_length:
          type: integer
          description: Max chars
          required: false
    hints:
      use_when: Read one page in detail.
      avoid_when: Do not use for mutations.
      arg_rules: url is required.
`
	require.NoError(t, os.WriteFile(overridePath, []byte(overrideYAML), 0o600))

	registry := newMCPRegistryWithConnector(
		Config{
			Endpoint:      "http://localhost:8811/mcp",
			ToolOverrides: overridePath,
		},
		&fakeConnector{session: session},
		nil,
		"",
	)

	require.NoError(t, registry.initializeActions(context.Background()))
	defs := registry.ListEmbeddings(context.Background())
	require.Len(t, defs, 1)

	def := defs[0].Action.Definition()
	assert.Equal(t, "fetch", def.Name)
	assert.Equal(t, "Fetch web content with constraints", def.Description)
	assert.Equal(t, "object", def.Input.Type)
	assert.Equal(t, "string", def.Input.Fields["url"].Type)
	assert.Equal(t, "Absolute URL", def.Input.Fields["url"].Description)
	assert.True(t, def.Input.Fields["url"].Required)
	assert.Equal(t, "integer", def.Input.Fields["max_length"].Type)
	assert.Equal(t, "Read one page in detail.", def.Hints.UseWhen)
	assert.Equal(t, "Do not use for mutations.", def.Hints.AvoidWhen)
	assert.Equal(t, "url is required.", def.Hints.ArgRules)
}
