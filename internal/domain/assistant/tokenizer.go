package assistant

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// Tokenizer estimates token counts for a model-specific input payload.
type Tokenizer interface {
	CountTokens(ctx context.Context, model string, input string) (int, error)
}

// BuildChatMessageTokenizationInput serializes the persisted parts of a chat message
// that contribute to active conversation context.
func BuildChatMessageTokenizationInput(message ChatMessage) string {
	var payload strings.Builder

	payload.WriteString(string(message.ChatRole))
	payload.WriteString("\n")
	payload.WriteString(message.Content)

	if message.ActionCallID != nil {
		payload.WriteString("\n")
		payload.WriteString(*message.ActionCallID)
	}
	if len(message.ActionCalls) > 0 {
		if raw, err := json.Marshal(message.ActionCalls); err == nil {
			payload.WriteString("\n")
			payload.Write(raw)
		}
	}

	return payload.String()
}

// EstimateTokenCountFallback approximates a token count when no model-specific tokenizer is available.
func EstimateTokenCountFallback(input string) int {
	chars := utf8.RuneCountInString(strings.TrimSpace(input))
	if chars == 0 {
		return 0
	}

	return (chars + 3) / 4
}
