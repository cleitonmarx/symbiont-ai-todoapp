package modelrunner

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
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

func (a gemmaEmbedding) GenerateIndexingPrompt(todo domain.Todo) string {
	return fmt.Sprintf("title: none | text: %s", todo.Title)
}

func (a gemmaEmbedding) GenerateSearchPrompt(searchInput string) string {
	return fmt.Sprintf("task: search result | query: %s", searchInput)
}

func (a gemmaEmbedding) GenerateAssistentActionDefinitionPrompt(action domain.AssistantActionDefinition) string {
	title := common.NormalizeWhitespace(action.Name)
	if title == "" {
		title = "none"
	}

	parts := []string{
		fmt.Sprintf("action: %s", common.NormalizeWhitespace(strings.ReplaceAll(action.Name, "_", " "))),
		fmt.Sprintf("description: %s", common.NormalizeWhitespace(action.Description)),
	}

	if action.HasHints() {
		parts = append(parts, fmt.Sprintf("hints: %s", common.NormalizeWhitespace(action.ComposeHint())))
	}

	if action.Input.Type != "" || len(action.Input.Fields) > 0 {
		parts = append(parts, fmt.Sprintf("input_type: %s", common.NormalizeWhitespace(action.Input.Type)))
	}

	if len(action.Input.Fields) > 0 {
		fieldNames := make([]string, 0, len(action.Input.Fields))
		for name := range action.Input.Fields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)

		fieldParts := make([]string, 0, len(fieldNames))
		for _, name := range fieldNames {
			field := action.Input.Fields[name]
			required := "optional"
			if field.Required {
				required = "required"
			}
			fieldParts = append(
				fieldParts,
				fmt.Sprintf(
					"%s(%s,%s): %s",
					common.NormalizeWhitespace(name),
					common.NormalizeWhitespace(field.Type),
					required,
					common.NormalizeWhitespace(field.Description),
				),
			)
		}
		parts = append(parts, fmt.Sprintf("fields: %s", strings.Join(fieldParts, "; ")))
	}

	text := strings.Join(parts, " | ")
	return fmt.Sprintf("title: %s | text: %s", title, text)
}

func (a gemmaEmbedding) Dimensions() *int {
	return nil
}

// defaultEmbeddingGenerator is a fallback implementation of EmbeddingGenerator
// that generates simple prompts without model-specific formatting.
type defaultEmbeddingGenerator struct{}

func (a defaultEmbeddingGenerator) GenerateIndexingPrompt(todo domain.Todo) string {
	return fmt.Sprintf("Item: %s. Due date: %s. Current status: %s.", todo.Title, todo.DueDate.Format(time.DateOnly), todo.Status)
}

func (a defaultEmbeddingGenerator) GenerateSearchPrompt(searchInput string) string {
	return searchInput
}

func (a defaultEmbeddingGenerator) GenerateAssistentActionDefinitionPrompt(action domain.AssistantActionDefinition) string {
	return fmt.Sprintf("Action Name: %s. Description: %s.", action.Name, action.Description)
}

func (a defaultEmbeddingGenerator) Dimensions() *int {
	return common.Ptr(768)
}
