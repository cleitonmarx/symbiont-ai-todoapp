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

func (f embeddingFactory) Get(model string) EmbeddingGenerator {
	if strings.Contains(model, "embeddinggemma") {
		return gemmaEmbedding{}
	}
	return defaultEmbeddingGenerator{}
}

// gemmaEmbedding implements the EmbeddingGenerator interface for the Gemma embedding model.
type gemmaEmbedding struct{}

func (a gemmaEmbedding) GenerateIndexingPrompt(document string) string {
	return fmt.Sprintf("title: none | text: %s", document)
}

func (a gemmaEmbedding) GenerateSkillPrompt(title, document string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "none"
	}
	return fmt.Sprintf("title: %s | text: %s", title, document)
}

func (a gemmaEmbedding) GenerateSearchPrompt(searchInput string) string {
	return fmt.Sprintf("task: search result | query: %s", searchInput)
}

func (a gemmaEmbedding) Dimensions() *int {
	return nil
}

// defaultEmbeddingGenerator is a fallback implementation of EmbeddingGenerator
// that generates simple prompts without model-specific formatting.
type defaultEmbeddingGenerator struct{}

func (a defaultEmbeddingGenerator) GenerateIndexingPrompt(document string) string {
	return document
}

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

func (a defaultEmbeddingGenerator) GenerateSearchPrompt(searchInput string) string {
	return searchInput
}

func (a defaultEmbeddingGenerator) Dimensions() *int {
	return common.Ptr(768)
}
