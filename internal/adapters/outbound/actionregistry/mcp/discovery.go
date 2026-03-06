package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/toon-format/toon-go"
)

// listAllTools paginates through MCP ListTools until no cursor is returned.
func listAllTools(ctx context.Context, session mcpSession) ([]*mcp.Tool, error) {
	tools := make([]*mcp.Tool, 0)
	cursor := ""

	for {
		res, err := session.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		if res == nil {
			return tools, nil
		}

		tools = append(tools, res.Tools...)
		next := strings.TrimSpace(res.NextCursor)
		if next == "" || next == cursor {
			return tools, nil
		}
		cursor = next
	}
}

// parseActionCallArguments validates assistant tool input and guarantees a JSON object payload.
func parseActionCallArguments(input string) (map[string]any, error) {
	if strings.TrimSpace(input) == "" {
		return map[string]any{}, nil
	}

	decoder := json.NewDecoder(strings.NewReader(input))
	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}

	args, ok := payload.(map[string]any)
	if !ok {
		return nil, errors.New("action arguments must be a JSON object")
	}
	return args, nil
}

// renderCallToolResult flattens MCP call output into plain text for tool messages.
func renderCallToolResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	if result.StructuredContent != nil {
		if bytes, err := toon.Marshal(result.StructuredContent); err == nil {
			return string(bytes)
		}
	}

	parts := make([]string, 0, len(result.Content)+1)
	for _, content := range result.Content {
		text := strings.TrimSpace(renderContent(content))
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n")
	}

	return ""
}

// renderContent converts one MCP content variant to a user-facing string representation.
func renderContent(content mcp.Content) string {
	switch item := content.(type) {
	case *mcp.TextContent:
		return item.Text
	case *mcp.ImageContent:
		return fmt.Sprintf("[image mime_type=%s bytes=%d]", item.MIMEType, len(item.Data))
	case *mcp.AudioContent:
		return fmt.Sprintf("[audio mime_type=%s bytes=%d]", item.MIMEType, len(item.Data))
	case *mcp.ResourceLink:
		return fmt.Sprintf("[resource_link uri=%s name=%s]", item.URI, item.Name)
	case *mcp.EmbeddedResource:
		if item.Resource == nil {
			return "[embedded_resource]"
		}
		if item.Resource.Text != "" {
			return item.Resource.Text
		}
		if len(item.Resource.Blob) > 0 {
			return fmt.Sprintf("[embedded_resource_blob uri=%s bytes=%d]", item.Resource.URI, len(item.Resource.Blob))
		}
		return fmt.Sprintf("[embedded_resource uri=%s]", item.Resource.URI)
	default:
		bytes, err := json.Marshal(item)
		if err != nil {
			return fmt.Sprintf("%v", item)
		}
		return string(bytes)
	}
}

// actionErrorMessage formats a structured tool error payload consumed by the assistant loop.
func actionErrorMessage(callID, code, details string) assistant.Message {
	return assistant.Message{
		Role:         assistant.ChatRole_Tool,
		ActionCallID: common.Ptr(callID),
		Content:      fmt.Sprintf("errors[1]{error,details}%s,%s", code, details),
	}
}
