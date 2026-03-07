package mcp

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseActionCallArguments(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		want    map[string]any
		wantErr string
	}{
		"empty-input": {
			input: "",
			want:  map[string]any{},
		},
		"valid-object": {
			input: `{"query":"hello","max_results":2}`,
			want:  map[string]any{"query": "hello", "max_results": float64(2)},
		},
		"invalid-json": {
			input:   `{"query":`,
			wantErr: "unexpected EOF",
		},
		"non-object-json": {
			input:   `["a"]`,
			wantErr: "action arguments must be a JSON object",
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
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

func TestRenderCallToolResult(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		result *mcp.CallToolResult
		assert func(*testing.T, string)
	}{
		"nil-result": {
			result: nil,
			assert: func(t *testing.T, got string) { assert.Equal(t, "", got) },
		},
		"structured-content": {
			result: &mcp.CallToolResult{StructuredContent: map[string]any{"k": "v"}},
			assert: func(t *testing.T, got string) {
				assert.Contains(t, got, "k")
				assert.Contains(t, got, "v")
			},
		},
		"text-content-joins-non-empty-lines": {
			result: &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "line1"}, &mcp.TextContent{Text: "   "}, &mcp.TextContent{Text: "line2"}}},
			assert: func(t *testing.T, got string) { assert.Equal(t, "line1\nline2", got) },
		},
		"resource-link-content": {
			result: &mcp.CallToolResult{Content: []mcp.Content{&mcp.ResourceLink{URI: "https://example.com", Name: "example"}}},
			assert: func(t *testing.T, got string) {
				assert.Contains(t, got, "resource_link")
				assert.Contains(t, got, "https://example.com")
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, renderCallToolResult(tt.result))
		})
	}
}

func TestRenderContent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content mcp.Content
		assert  func(*testing.T, string)
	}{
		"text": {
			content: &mcp.TextContent{Text: "hello"},
			assert:  func(t *testing.T, got string) { assert.Equal(t, "hello", got) },
		},
		"image": {
			content: &mcp.ImageContent{MIMEType: "image/png", Data: []byte("abcd")},
			assert:  func(t *testing.T, got string) { assert.Contains(t, got, "image/png") },
		},
		"audio": {
			content: &mcp.AudioContent{MIMEType: "audio/wav", Data: []byte("abcd")},
			assert:  func(t *testing.T, got string) { assert.Contains(t, got, "audio/wav") },
		},
		"embedded-resource-text": {
			content: &mcp.EmbeddedResource{Resource: &mcp.ResourceContents{Text: "embedded", URI: "file://a"}},
			assert:  func(t *testing.T, got string) { assert.Equal(t, "embedded", got) },
		},
		"embedded-resource-blob": {
			content: &mcp.EmbeddedResource{Resource: &mcp.ResourceContents{Blob: []byte("abcd"), URI: "file://a"}},
			assert:  func(t *testing.T, got string) { assert.Contains(t, got, "embedded_resource_blob") },
		},
		"embedded-resource-nil": {
			content: &mcp.EmbeddedResource{},
			assert:  func(t *testing.T, got string) { assert.Equal(t, "[embedded_resource]", got) },
		},
		"fallback-json": {
			content: &mcp.ResourceLink{URI: "https://example.com", Name: "example"},
			assert:  func(t *testing.T, got string) { assert.NotEmpty(t, got) },
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, renderContent(tt.content))
		})
	}
}

func TestListAllTools(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		session *fakeSession
		assert  func(*testing.T, []*mcp.Tool, error)
	}{
		"nil-response-returns-empty": {
			session: &fakeSession{listResults: []*mcp.ListToolsResult{nil}},
			assert: func(t *testing.T, got []*mcp.Tool, err error) {
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		"multi-page-results": {
			session: &fakeSession{listResults: []*mcp.ListToolsResult{{Tools: []*mcp.Tool{{Name: "search"}}, NextCursor: "cursor-1"}, {Tools: []*mcp.Tool{{Name: "fetch_content"}}}}},
			assert: func(t *testing.T, got []*mcp.Tool, err error) {
				require.NoError(t, err)
				require.Len(t, got, 2)
				assert.Equal(t, "search", got[0].Name)
				assert.Equal(t, "fetch_content", got[1].Name)
			},
		},
		"list-error": {
			session: &fakeSession{listErr: assert.AnError},
			assert: func(t *testing.T, _ []*mcp.Tool, err error) {
				require.ErrorIs(t, err, assert.AnError)
			},
		},
		"repeated-cursor-stops": {
			session: &fakeSession{listResults: []*mcp.ListToolsResult{{Tools: []*mcp.Tool{{Name: "search"}}, NextCursor: "cursor-1"}, {Tools: []*mcp.Tool{{Name: "fetch_content"}}, NextCursor: "cursor-1"}}},
			assert: func(t *testing.T, got []*mcp.Tool, err error) {
				require.NoError(t, err)
				require.Len(t, got, 2)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := listAllTools(t.Context(), tt.session)
			tt.assert(t, got, err)
		})
	}
}

func TestActionErrorMessage(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		callID  string
		code    string
		details string
	}{
		"formats-error-payload": {callID: "call-1", code: "invalid_arguments", details: "bad json"},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			msg := actionErrorMessage(tt.callID, tt.code, tt.details)
			assert.Equal(t, assistant.ChatRole_Tool, msg.Role)
			require.NotNil(t, msg.ActionCallID)
			assert.Equal(t, tt.callID, *msg.ActionCallID)
			assert.Contains(t, msg.Content, tt.code)
			assert.Contains(t, msg.Content, tt.details)
		})
	}
}
