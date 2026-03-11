package modelrunner

import (
	"context"
	"errors"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// SemanticEncoder implements the semantic.Encoder interface using a DRMAPIClient
type SemanticEncoder struct {
	embeddingClient  DRMAPIClient
	embeddingFactory EmbeddingFactory
}

// NewSemanticEncoder creates a new SemanticEncoder with the provided DRMAPIClient for embeddings and an EmbeddingFactory for prompt generation.
func NewSemanticEncoder(client DRMAPIClient) SemanticEncoder {
	return SemanticEncoder{
		embeddingClient:  client,
		embeddingFactory: embeddingFactory{},
	}
}

// VectorizeTodo implements semantic.Encoder.VectorizeTodo.
func (a SemanticEncoder) VectorizeTodo(ctx context.Context, model string, todo todo.Todo) (semantic.EmbeddingVector, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()
	gen := a.embeddingFactory.Get(model)
	prompt := gen.GenerateIndexingPrompt(todo.Title)
	dimension := gen.Dimensions()
	vec, err := a.embed(spanCtx, model, prompt, dimension)
	if telemetry.IsErrorRecorded(span, err) {
		return semantic.EmbeddingVector{}, err
	}
	return vec, nil
}

// VectorizeQuery implements semantic.Encoder.VectorizeQuery.
func (a SemanticEncoder) VectorizeQuery(ctx context.Context, model, query string) (semantic.EmbeddingVector, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	gen := a.embeddingFactory.Get(model)
	prompt := gen.GenerateSearchPrompt(query)
	dimension := gen.Dimensions()
	vec, err := a.embed(spanCtx, model, prompt, dimension)
	if telemetry.IsErrorRecorded(span, err) {
		return semantic.EmbeddingVector{}, err
	}
	return vec, nil
}

// VectorizeSkillDefinition implements semantic.Encoder.VectorizeSkillDefinition.
func (a SemanticEncoder) VectorizeSkillDefinition(
	ctx context.Context,
	model string,
	skill assistant.SkillDefinition,
) (semantic.EmbeddingVector, semantic.EmbeddingVector, error) {
	gen := a.embeddingFactory.Get(model)
	dimension := gen.Dimensions()
	var (
		useVector semantic.EmbeddingVector
		err       error
	)
	if strings.TrimSpace(skill.UseWhen) != "" {
		useText := gen.GenerateSkillPrompt(skill.Name, buildSkillUseEmbeddingText(skill))
		useVector, err = a.embed(ctx, model, useText, dimension)
		if err != nil {
			return semantic.EmbeddingVector{}, semantic.EmbeddingVector{}, err
		}
	}

	var avoidVector semantic.EmbeddingVector
	if strings.TrimSpace(skill.AvoidWhen) != "" {
		avoidText := gen.GenerateSkillPrompt(skill.Name, buildSkillAvoidEmbeddingText(skill))
		avoidVector, err = a.embed(ctx, model, avoidText, dimension)
		if err != nil {
			return semantic.EmbeddingVector{}, semantic.EmbeddingVector{}, err
		}
	}
	return useVector, avoidVector, nil
}

// buildSkillUseEmbeddingText constructs the text to be embedded
// for a skill's "use" conditions, including the main useWhen text,
// an optional first line of the content, tags, and tools.
func buildSkillUseEmbeddingText(skill assistant.SkillDefinition) string {
	parts := make([]string, 0, 5)
	parts = appendIfNotEmpty(parts, strings.TrimSpace(skill.UseWhen))
	if skill.EmbedFirstContentLine {
		parts = appendIfNotEmpty(parts, firstSkillContentLine(skill.Content))
	}
	if len(skill.Tags) > 0 {
		parts = append(parts, "Related terms: "+strings.Join(skill.Tags, ", "))
	}
	if len(skill.Tools) > 0 {
		parts = append(parts, "Actions/tools: "+strings.Join(skill.Tools, ", "))
	}
	return strings.Join(parts, "\n")
}

// buildSkillAvoidEmbeddingText constructs the text to be embedded for a
// skill's "avoid" conditions.
func buildSkillAvoidEmbeddingText(skill assistant.SkillDefinition) string {
	avoid := strings.TrimSpace(skill.AvoidWhen)
	if avoid == "" {
		return ""
	}
	return "Avoid when: " + avoid
}

// appendIfNotEmpty appends a string to a slice if the string is not empty or whitespace.
func appendIfNotEmpty(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	return append(values, value)
}

// firstSkillContentLine extracts the first non-empty line from the skill content,
// if EmbedFirstContentLine is true.
func firstSkillContentLine(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return line
	}
	return ""
}

// embed sends a request to the embedding API and returns the resulting vector in a provider-agnostic shape.
func (a SemanticEncoder) embed(ctx context.Context, model, input string, dimension *int) (semantic.EmbeddingVector, error) {
	req := EmbeddingsRequest{Model: model, Input: input, Dimensions: dimension}
	resp, err := a.embeddingClient.Embeddings(ctx, req)
	if err != nil {
		return semantic.EmbeddingVector{}, err
	}
	if len(resp.Data) == 0 {
		return semantic.EmbeddingVector{}, errors.New("no embedding data in response")
	}
	return semantic.EmbeddingVector{
		Vector:      resp.Data[0].Embedding,
		TotalTokens: resp.Usage.TotalTokens,
	}, nil
}
