package assistant

import "context"

// SkillDefinition describes one static skill document that can be injected in prompts.
type SkillDefinition struct {
	Name                  string
	DisplayName           string
	Aliases               []string
	Description           string
	UseWhen               string
	AvoidWhen             string
	Priority              int
	Tags                  []string
	Tools                 []string
	EmbedFirstContentLine bool
	Content               string
	Source                string
}

// SelectedSkill describes a skill selected for use in a turn, including any tools to call.
type SelectedSkill struct {
	Name   string
	Source string
	Tools  []string
}

// SkillQueryContext carries turn context used for skill relevance scoring.
type SkillQueryContext struct {
	Messages            []Message
	ConversationSummary string
}

// SkillRegistry resolves relevant skills based on user input.
type SkillRegistry interface {
	// ListRelevant returns relevant skill definitions for the current turn.
	ListRelevant(ctx context.Context, query SkillQueryContext) []SkillDefinition
	// ListSkills returns the full set of registered skill definitions.
	ListSkills(ctx context.Context) ([]SkillDefinition, error)
}

// NewSelectedSkill creates an SelectedSkill from a definition.
func NewSelectedSkill(def SkillDefinition) SelectedSkill {
	tools := make([]string, len(def.Tools))
	copy(tools, def.Tools)

	return SelectedSkill{
		Name:   def.Name,
		Source: def.Source,
		Tools:  tools,
	}
}
