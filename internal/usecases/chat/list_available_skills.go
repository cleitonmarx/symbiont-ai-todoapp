package chat

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// ListAvailableSkills defines the use case for listing available assistant skills.
type ListAvailableSkills interface {
	Query(ctx context.Context) ([]assistant.SkillDefinition, error)
}

// ListAvailableSkillsImpl implements the ListAvailableSkills use case.
type ListAvailableSkillsImpl struct {
	skillRegistry assistant.SkillRegistry
}

// NewListAvailableSkillsImpl creates a new ListAvailableSkillsImpl instance.
func NewListAvailableSkillsImpl(skillRegistry assistant.SkillRegistry) *ListAvailableSkillsImpl {
	return &ListAvailableSkillsImpl{skillRegistry: skillRegistry}
}

// Query retrieves the list of available skills from the configured catalog.
func (uc ListAvailableSkillsImpl) Query(ctx context.Context) ([]assistant.SkillDefinition, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	skills, err := uc.skillRegistry.ListSkills(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, err
	}

	return skills, nil
}
