package modelrunner

import (
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// EmbeddingGenerator defines the interface for generating embeddings for todos and search inputs.
type EmbeddingGenerator interface {
	// GenerateIndexingPrompt creates the prompt used for generating embeddings for a todo item.
	GenerateIndexingPrompt(todo domain.Todo) string
	// GenerateSearchPrompt creates the prompt used for generating embeddings for a search query.
	GenerateSearchPrompt(searchInput string) string
	// GenerateAssistentActionDefinitionPrompt creates the prompt used for generating embeddings for an assistant action.
	GenerateAssistentActionDefinitionPrompt(action domain.AssistantActionDefinition) string
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

func (a gemmaEmbedding) GenerateIndexingPrompt(todo domain.Todo) string {
	return fmt.Sprintf("title: none | text: %s", todo.Title)
}

func (a gemmaEmbedding) GenerateSearchPrompt(searchInput string) string {
	return fmt.Sprintf("task: search result | query: %s", searchInput)
}

func (a gemmaEmbedding) GenerateAssistentActionDefinitionPrompt(action domain.AssistantActionDefinition) string {
	return fmt.Sprintf("title: %s | text: %s", action.Name, action.Description)
}

// defaultEmbeddingGenerator is a fallback implementation of EmbeddingGenerator
// that generates simple prompts without model-specific formatting.
type defaultEmbeddingGenerator struct{}

func (a defaultEmbeddingGenerator) GenerateIndexingPrompt(todo domain.Todo) string {
	return fmt.Sprintf("title:'%s'\ndue_date:'%s'\nstatus:'%s'", todo.Title, todo.DueDate.Format(time.RFC3339), todo.Status)
}

func (a defaultEmbeddingGenerator) GenerateSearchPrompt(searchInput string) string {
	return searchInput
}

func (a defaultEmbeddingGenerator) GenerateAssistentActionDefinitionPrompt(action domain.AssistantActionDefinition) string {
	return fmt.Sprintf("title:'%s'\ndescription:'%s'", action.Name, action.Description)
}
