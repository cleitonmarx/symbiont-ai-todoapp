package modelrunner

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestGemmaEmbedding_GenerateAssistentActionDefinitionPrompt(t *testing.T) {
	t.Parallel()

	embedding := gemmaEmbedding{}

	action := domain.AssistantActionDefinition{
		Name:        "create_todos",
		Description: "Create multiple todo items in one call.",
		Hints: domain.AssistantActionHints{
			UseWhen:   "Use for checklist and plan prompts.",
			AvoidWhen: "Do not use for single-item operations.",
			ArgRules:  "Required key: todos.",
		},
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"todos": {
					Type:        "array",
					Required:    true,
					Description: "List of todos.",
				},
				"title_prefix": {
					Type:        "string",
					Required:    false,
					Description: "Optional prefix.",
				},
			},
		},
	}

	got := embedding.GenerateAssistentActionDefinitionPrompt(action)

	expected := "title: create_todos | text: " +
		"action: create todos | " +
		"description: Create multiple todo items in one call. | " +
		"hints: Use: Use for checklist and plan prompts. Avoid: Do not use for single-item operations. Args: Required key: todos. | " +
		"input_type: object | " +
		"fields: title_prefix(string,optional): Optional prefix.; todos(array,required): List of todos."
	assert.Equal(t, expected, got)
}
