package domain

import "context"

// AssistantSkillDefinition describes one static skill document that can be injected in prompts.
type AssistantSkillDefinition struct {
	Name      string
	UseWhen   string
	AvoidWhen string
	Priority  int
	Tags      []string
	Tools     []string
	Content   string
	Source    string
}

// AssistantSkillRegistry resolves relevant skills based on user input.
type AssistantSkillRegistry interface {
	// ListRelevant returns relevant skill definitions for the current turn.
	ListRelevant(ctx context.Context, userInput string) []AssistantSkillDefinition
}
