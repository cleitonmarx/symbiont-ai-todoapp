package usecases

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter         = otel.Meter("usecases")
	LLMTokensUsed metric.Int64Counter
)

func init() {
	var err error
	// Tokens consumed by LLM (input + output)
	LLMTokensUsed, err = meter.Int64Counter(
		"llm_tokens_used_total",
		metric.WithDescription("Total LLM tokens consumed"),
	)
	if err != nil {
		panic(err)
	}
}

// RecordLLMTokensUsed records the number of tokens used in an LLM chat operation.
func RecordLLMTokensUsed(ctx context.Context, promptTokens, completionTokens int) {
	LLMTokensUsed.Add(ctx, int64(promptTokens), metric.WithAttributes(
		attribute.String("token_type", "prompt"),
	))
	LLMTokensUsed.Add(ctx, int64(completionTokens), metric.WithAttributes(
		attribute.String("token_type", "completion"),
	))
}

// RecordLLMTokensEmbedding records the number of tokens used in an embedding operation.
func RecordLLMTokensEmbedding(ctx context.Context, totalTokens int) {
	LLMTokensUsed.Add(ctx, int64(totalTokens), metric.WithAttributes(
		attribute.String("token_type", "embedding"),
	))
}
