package domain

import "context"

// AssistantSkillDefinition describes one static skill document that can be injected in prompts.
type AssistantSkillDefinition struct {
	Name                  string
	UseWhen               string
	AvoidWhen             string
	Priority              int
	Tags                  []string
	Tools                 []string
	EmbedFirstContentLine bool
	Content               string
	Source                string
}

// AssistantSelectedSkill describes a skill selected for use in a turn, including any tools to call.
type AssistantSelectedSkill struct {
	Name   string
	Source string
	Tools  []string
}

// AssistantSkillQueryContext carries turn context used for skill relevance scoring.
type AssistantSkillQueryContext struct {
	Messages            []AssistantMessage
	ConversationSummary string
}

// AssistantSkillRegistry resolves relevant skills based on user input.
type AssistantSkillRegistry interface {
	// ListRelevant returns relevant skill definitions for the current turn.
	ListRelevant(ctx context.Context, query AssistantSkillQueryContext) []AssistantSkillDefinition
}

// NewAssistantSelectedSkill creates an AssistantSelectedSkill from a definition.
func NewAssistantSelectedSkill(def AssistantSkillDefinition) AssistantSelectedSkill {
	tools := make([]string, len(def.Tools))
	copy(tools, def.Tools)

	return AssistantSelectedSkill{
		Name:   def.Name,
		Source: def.Source,
		Tools:  tools,
	}
}
