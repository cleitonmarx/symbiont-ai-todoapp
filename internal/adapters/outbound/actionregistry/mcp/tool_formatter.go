package mcp

import (
	"encoding/json"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// toolFormatter customizes MCP tool arguments before execution and/or formats
// tool results after execution.
type toolFormatter interface {
	FormatArguments(arguments map[string]any) map[string]any
	FormatResult(actionResult string, call domain.AssistantActionCall) domain.AssistantMessage
}

// executeCodeToolFormatter handles both the input and output quirks of the
// "execute_code" tool.
type executeCodeToolFormatter struct{}

func (f executeCodeToolFormatter) FormatArguments(arguments map[string]any) map[string]any {
	if len(arguments) == 0 {
		return arguments
	}

	code, ok := arguments["code"].(string)
	if !ok {
		return arguments
	}

	// Small models sometimes send Python as one JSON string with literal escape
	// sequences like "\n" instead of real newlines. Normalize only when the
	// script is single-line to avoid rewriting intentional escape sequences in
	// valid multi-line code.
	if strings.Contains(code, "\n") {
		return arguments
	}
	if !strings.Contains(code, `\n`) && !strings.Contains(code, `\t`) && !strings.Contains(code, `\r`) {
		return arguments
	}

	arguments["code"] = strings.NewReplacer(
		`\r\n`, "\n",
		`\n`, "\n",
		`\r`, "\r",
		`\t`, "\t",
	).Replace(code)

	return arguments
}

func (f executeCodeToolFormatter) FormatResult(actionResult string, call domain.AssistantActionCall) domain.AssistantMessage {
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

// toolFormatterRegistry maps tool names to their corresponding formatters and
// provides a single entry point for tool-specific argument/result handling.
type toolFormatterRegistry map[string]toolFormatter

// FormatArguments applies a registered argument formatter for the given tool name.
func (r toolFormatterRegistry) FormatArguments(toolName string, arguments map[string]any) (map[string]any, bool) {
	formatter, found := r[toolName]
	if !found {
		return arguments, false
	}
	return formatter.FormatArguments(arguments), true
}

// FormatResult applies a registered result formatter for the given tool name.
func (r toolFormatterRegistry) FormatResult(toolName, actionResult string, call domain.AssistantActionCall) (domain.AssistantMessage, bool) {
	formatter, found := r[toolName]
	if !found {
		return domain.AssistantMessage{}, false
	}
	return formatter.FormatResult(actionResult, call), true
}

// toolFormatters maps tool names to their corresponding formatters, allowing
// for custom handling of both arguments and results based on the tool name.
var toolFormatters = toolFormatterRegistry{
	"execute_code": executeCodeToolFormatter{},
}
