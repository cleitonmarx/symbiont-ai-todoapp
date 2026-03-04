package mcp

import (
	"encoding/json"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// resultFormatter formats the result of an MCP tool call.
type resultFormatter interface {
	Format(actionResult string, call domain.AssistantActionCall) domain.AssistantMessage
}

// executeCodeFormatter formats the action result of an "execute_code" tool call,
// extracting errors and results to create a structured response.
type executeCodeFormatter struct{}

func (f executeCodeFormatter) Format(actionResult string, call domain.AssistantActionCall) domain.AssistantMessage {
	var (
		result struct {
			Errors []string `json:"error"`
			Result []string `json:"result"`
		}
		content string
	)

	_ = json.Unmarshal([]byte(actionResult), &result) //nolint:errcheck
	if len(result.Errors) > 0 {
		content = `{"error":"code_error","details":"` + strings.Join(result.Errors, ", ") + `"}`
	} else {
		content = strings.Join(result.Result, "\n")
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      content,
	}
}

// resultFormatters maps tool names to their corresponding result formatters,
// allowing for custom formatting of action results based on the tool that was called.
var resultFormatters = map[string]resultFormatter{
	"execute_code": executeCodeFormatter{},
}
