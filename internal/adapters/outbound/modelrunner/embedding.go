package modelrunner

import (
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
)

// EmbeddingGenerator defines the interface for generating embeddings for todos and search inputs.
type EmbeddingGenerator interface {
	// GenerateIndexingPrompt creates the prompt used for generating embeddings for a todo item.
	GenerateIndexingPrompt(document string) string
	// GenerateSkillPrompt creates the prompt used for generating embeddings for a skill document.
	GenerateSkillPrompt(title, document string) string
	// GenerateSearchPrompt creates the prompt used for generating embeddings for a search query.
	GenerateSearchPrompt(searchInput string) string
	// Dimension returns the dimension used by the embedding generator, if applicable.
	Dimensions() *int
}

// EmbeddingFactory provides a method to get an EmbeddingGenerator based on the model name.
type EmbeddingFactory interface {
	// Get returns an EmbeddingGenerator for the specified model name.
	Get(model string) EmbeddingGenerator
}

// embeddingFactory is the default implementation of EmbeddingFactory.
type embeddingFactory struct {
}

// Get returns an embedding generator based on the model identifier.
func (f embeddingFactory) Get(model string) EmbeddingGenerator {
	if strings.Contains(model, "embeddinggemma") {
		return gemmaEmbedding{}
	}
	return defaultEmbeddingGenerator{}
}

// gemmaEmbedding implements the EmbeddingGenerator interface for the Gemma embedding model.
type gemmaEmbedding struct{}

// GenerateIndexingPrompt returns the Gemma-specific indexing prompt.
func (a gemmaEmbedding) GenerateIndexingPrompt(document string) string {
	return fmt.Sprintf("title: none | text: %s", document)
}

// GenerateSkillPrompt returns the Gemma-specific skill embedding prompt.
func (a gemmaEmbedding) GenerateSkillPrompt(title, document string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "none"
	}
	return fmt.Sprintf("title: %s | text: %s", title, document)
}

// GenerateSearchPrompt returns the Gemma-specific search embedding prompt.
func (a gemmaEmbedding) GenerateSearchPrompt(searchInput string) string {
	return fmt.Sprintf("task: search result | query: %s", searchInput)
}

// Dimensions returns the embedding dimensions for Gemma models.
func (a gemmaEmbedding) Dimensions() *int {
	return nil
}

// defaultEmbeddingGenerator is a fallback implementation of EmbeddingGenerator
// that generates simple prompts without model-specific formatting.
type defaultEmbeddingGenerator struct{}

// GenerateIndexingPrompt returns the default indexing prompt.
func (a defaultEmbeddingGenerator) GenerateIndexingPrompt(document string) string {
	return document
}

// GenerateSkillPrompt returns the default skill embedding prompt.
func (a defaultEmbeddingGenerator) GenerateSkillPrompt(title, document string) string {
	title = strings.TrimSpace(title)
	document = strings.TrimSpace(document)
	if title == "" {
		return document
	}
	if document == "" {
		return title
	}
	return title + "\n" + document
}

// GenerateSearchPrompt returns the default search embedding prompt.
func (a defaultEmbeddingGenerator) GenerateSearchPrompt(searchInput string) string {
	return searchInput
}

// Dimensions returns the embedding dimensions for default models.
func (a defaultEmbeddingGenerator) Dimensions() *int {
	return common.Ptr(768)
}
