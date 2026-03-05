package mcp

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPRegistry_Execute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		session       *fakeSession
		call          domain.AssistantActionCall
		initialize    bool
		assertMessage func(*testing.T, domain.AssistantMessage)
		assertSession func(*testing.T, *fakeSession)
	}{
		"calls-tool": {
			session:    &fakeSession{listResults: []*mcp.ListToolsResult{{Tools: []*mcp.Tool{{Name: "fetch", Description: "Fetches content", InputSchema: map[string]any{"type": "object"}}}}}, callResult: &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "done"}}}},
			call:       domain.AssistantActionCall{ID: "call-1", Name: "fetch", Input: `{"url":"https://example.com"}`},
			initialize: true,
			assertMessage: func(t *testing.T, msg domain.AssistantMessage) {
				require.NotNil(t, msg.ActionCallID)
				assert.Equal(t, "call-1", *msg.ActionCallID)
				assert.Equal(t, domain.ChatRole_Tool, msg.Role)
				assert.Equal(t, "done", msg.Content)
			},
			assertSession: func(t *testing.T, session *fakeSession) {
				require.NotNil(t, session.lastCallParams)
				assert.Equal(t, "fetch", session.lastCallParams.Name)
				assert.Equal(t, "https://example.com", session.lastCallParams.Arguments.(map[string]any)["url"])
			},
		},
		"invalid-arguments": {
			session:       &fakeSession{listResults: []*mcp.ListToolsResult{{Tools: []*mcp.Tool{{Name: "fetch", Description: "Fetches content", InputSchema: map[string]any{"type": "object"}}}}}},
			call:          domain.AssistantActionCall{ID: "call-2", Name: "fetch", Input: `[]`},
			initialize:    true,
			assertMessage: func(t *testing.T, msg domain.AssistantMessage) { assert.Contains(t, msg.Content, "invalid_arguments") },
		},
		"error-prefixes-content": {
			session:       &fakeSession{listResults: []*mcp.ListToolsResult{{Tools: []*mcp.Tool{{Name: "fetch", Description: "Fetches content", InputSchema: map[string]any{"type": "object"}}}}}, callResult: &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "failed"}}}},
			call:          domain.AssistantActionCall{ID: "call-3", Name: "fetch", Input: `{}`},
			initialize:    true,
			assertMessage: func(t *testing.T, msg domain.AssistantMessage) { assert.Equal(t, "error: failed", msg.Content) },
		},
		"execute-code-normalizes-escaped-newlines": {
			session: &fakeSession{
				listResults: []*mcp.ListToolsResult{{Tools: []*mcp.Tool{{Name: "execute_code", Description: "Executes code", InputSchema: map[string]any{"type": "object"}}}}},
				callResult:  &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: `{"result":["ok"]}`}}},
			},
			call:       domain.AssistantActionCall{ID: "call-exec", Name: "execute_code", Input: `{"code":"result = 1\\nresult"}`},
			initialize: true,
			assertMessage: func(t *testing.T, msg domain.AssistantMessage) {
				assert.Equal(t, "ok", msg.Content)
			},
			assertSession: func(t *testing.T, session *fakeSession) {
				require.NotNil(t, session.lastCallParams)
				assert.Equal(t, "execute_code", session.lastCallParams.Name)
				assert.Equal(t, "result = 1\nresult", session.lastCallParams.Arguments.(map[string]any)["code"])
			},
		},
		"unknown-action": {
			session:       &fakeSession{},
			call:          domain.AssistantActionCall{ID: "call-unknown", Name: "missing_tool", Input: `{}`},
			assertMessage: func(t *testing.T, msg domain.AssistantMessage) { assert.Contains(t, msg.Content, "unknown_action") },
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			registry := newMCPRegistryWithConnector(Config{Endpoint: "http://localhost:8811/mcp"}, &fakeConnector{session: tt.session})
			if tt.initialize {
				require.NoError(t, registry.initializeActions(t.Context()))
			}
			msg := registry.Execute(context.Background(), tt.call, nil)
			tt.assertMessage(t, msg)
			if tt.assertSession != nil {
				tt.assertSession(t, tt.session)
			}
		})
	}
}

func TestMCPRegistry_Methods(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		registry *MCPRegistry
		assert   func(*testing.T, *MCPRegistry)
	}{
		"get-definition-found": {
			registry: &MCPRegistry{actionsByName: map[string]domain.AssistantAction{"search": mcpToolAction{definition: domain.AssistantActionDefinition{Name: "search"}}}},
			assert: func(t *testing.T, registry *MCPRegistry) {
				def, found := registry.GetDefinition("search")
				require.True(t, found)
				assert.Equal(t, "search", def.Name)
			},
		},
		"get-renderer-found": {
			registry: &MCPRegistry{actionsByName: map[string]domain.AssistantAction{"execute_code": mcpToolAction{renderer: fakeRenderer{ok: true}}}},
			assert: func(t *testing.T, registry *MCPRegistry) {
				renderer, found := registry.GetRenderer("execute_code")
				require.True(t, found)
				assert.NotNil(t, renderer)
			},
		},
		"status-message-found": {
			registry: &MCPRegistry{actionsByName: map[string]domain.AssistantAction{"search": mcpToolAction{statusMessage: "Searching..."}}},
			assert: func(t *testing.T, registry *MCPRegistry) {
				assert.Equal(t, "Searching...", registry.StatusMessage("search"))
			},
		},
		"status-message-defaults": {
			registry: &MCPRegistry{},
			assert: func(t *testing.T, registry *MCPRegistry) {
				assert.Equal(t, defaultStatusMessage, registry.StatusMessage("missing"))
				assert.Equal(t, defaultStatusMessage, registry.StatusMessage(" "))
			},
		},
		"close-session": {
			registry: &MCPRegistry{session: &fakeSession{}},
			assert: func(t *testing.T, registry *MCPRegistry) {
				fake := registry.session.(*fakeSession)
				require.NoError(t, registry.Close())
				assert.Equal(t, 1, fake.closeCalls)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, tt.registry)
		})
	}
}

func TestRegistry_InitializeActions_AppliesToolOverrides(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tool   *mcp.Tool
		assert func(*testing.T, *MCPRegistry)
	}{
		"search": {
			tool: &mcp.Tool{Name: "search", Description: "Original description", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string", "description": "Query"}}, "required": []any{"query"}}},
			assert: func(t *testing.T, registry *MCPRegistry) {
				def, found := registry.GetDefinition("search")
				require.True(t, found)
				assert.Equal(t, "Search the web with DuckDuckGo and return concise result snippets with source links.", def.Description)
				assert.Equal(t, "Search query in natural language.", def.Input.Fields["query"].Description)
				assert.Equal(t, "🔎 Searching on the web...", registry.StatusMessage("search"))
			},
		},
		"execute-code": {
			tool: &mcp.Tool{Name: "execute_code", Description: "Original execute code description", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"code": map[string]any{"type": "string", "description": "Original code description"}, "session_id": map[string]any{"type": "integer", "description": "Original session_id description"}}, "required": []any{"code"}}},
			assert: func(t *testing.T, registry *MCPRegistry) {
				def, found := registry.GetDefinition("execute_code")
				require.True(t, found)
				assert.Equal(t, "Execute short self-contained Python code for deterministic calculations, grouping, validation, and data shaping. The tool input must be valid JSON, but the `code` field must contain raw Python source with real newlines, not escaped source like `\\\\n` or `\\\\t`. Prefer compact self-contained scripts and use single quotes inside Python when possible to reduce escaping.", def.Description)
				assert.Equal(t, "🧮 Running code...", registry.StatusMessage("execute_code"))
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			session := &fakeSession{listResults: []*mcp.ListToolsResult{{Tools: []*mcp.Tool{tt.tool}}}}
			registry := newMCPRegistryWithConnector(Config{Endpoint: "http://localhost:8811/mcp"}, &fakeConnector{session: session})
			require.NoError(t, registry.initializeActions(context.Background()))
			tt.assert(t, registry)
		})
	}
}
