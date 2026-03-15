package tokenizer

import (
	"testing"

	qwentokenizer "github.com/CharLemAznable/qwen-tokenizer"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	openaitokenizer "github.com/tiktoken-go/tokenizer"
)

func TestDefaultTokenizer_CountTokens(t *testing.T) {
	t.Parallel()

	t.Run("uses qwen tokenizer for qwen models", func(t *testing.T) {
		t.Parallel()

		input := "List overdue todos and summarize the blockers."
		expected := len((&qwentokenizer.Tokenizer{}).EncodeOrdinary(input))

		got, err := DefaultTokenizer{}.CountTokens(t.Context(), "qwen3:4B-F16", input)

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("uses tiktoken for openai models", func(t *testing.T) {
		t.Parallel()

		input := "Summarize the current sprint risks."
		encoding, err := openaitokenizer.ForModel("gpt-4o-mini")
		require.NoError(t, err)
		expectedTokens, _, err := encoding.Encode(input)
		require.NoError(t, err)

		got, err := DefaultTokenizer{}.CountTokens(t.Context(), "gpt-4o-mini", input)

		require.NoError(t, err)
		assert.Equal(t, len(expectedTokens), got)
	})

	t.Run("falls back for unknown models", func(t *testing.T) {
		t.Parallel()

		input := "unknown model token estimation fallback"

		got, err := DefaultTokenizer{}.CountTokens(t.Context(), "mistral-small", input)

		require.NoError(t, err)
		assert.Equal(t, assistant.EstimateTokenCountFallback(input), got)
	})
}
