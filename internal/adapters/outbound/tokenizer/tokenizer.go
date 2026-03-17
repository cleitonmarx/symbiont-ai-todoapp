package tokenizer

import (
	"context"
	"strings"

	qwentokenizer "github.com/CharLemAznable/qwen-tokenizer"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	openaitokenizer "github.com/tiktoken-go/tokenizer"
)

// DefaultTokenizer provides a model-aware contract with a heuristic fallback implementation.
type DefaultTokenizer struct{}

// CountTokens estimates the token count for the given model and input.
func (DefaultTokenizer) CountTokens(ctx context.Context, model string, input string) (int, error) {
	_, span := telemetry.StartSpan(ctx)
	defer span.End()

	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	switch {
	case isQwenModel(normalizedModel):
		if count, err := countQwenTokens(input); err == nil {
			return count, nil
		}
	case isOpenAIModel(normalizedModel):
		if count, err := countOpenAITokens(normalizedModel, input); err == nil {
			return count, nil
		}
	}

	return assistant.EstimateTokenCountFallback(input), nil
}

func isQwenModel(model string) bool {
	return strings.Contains(model, "qwen")
}

func isOpenAIModel(model string) bool {
	return strings.Contains(model, "gpt-") ||
		strings.Contains(model, "chatgpt") ||
		strings.HasPrefix(model, "o1") ||
		strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "o4") ||
		strings.Contains(model, "/o1") ||
		strings.Contains(model, "/o3") ||
		strings.Contains(model, "/o4")
}

func countQwenTokens(input string) (int, error) {
	tokenizer := &qwentokenizer.Tokenizer{}
	return len(tokenizer.EncodeOrdinary(input)), nil
}

func countOpenAITokens(model string, input string) (int, error) {
	encoding, err := openaitokenizer.ForModel(openaitokenizer.Model(model))
	if err != nil {
		encoding, err = openaitokenizer.Get(defaultOpenAIEncoding(model))
		if err != nil {
			return 0, err
		}
	}

	tokenIDs, _, err := encoding.Encode(input)
	if err != nil {
		return 0, err
	}

	return len(tokenIDs), nil
}

func defaultOpenAIEncoding(model string) openaitokenizer.Encoding {
	if strings.Contains(model, "gpt-4o") ||
		strings.Contains(model, "gpt-4.1") ||
		strings.Contains(model, "gpt-5") ||
		strings.Contains(model, "o1") ||
		strings.Contains(model, "o3") ||
		strings.Contains(model, "o4") {
		return openaitokenizer.O200kBase
	}

	return openaitokenizer.Cl100kBase
}
