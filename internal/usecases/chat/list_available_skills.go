package chat

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// ListAvailableSkills returns the assistant skills exposed to chat clients.
type ListAvailableSkills interface {
	// Query returns the currently available skills.
	Query(ctx context.Context) ([]assistant.SkillDefinition, error)
}

// ListAvailableSkillsImpl implements ListAvailableSkills.
type ListAvailableSkillsImpl struct {
	skillRegistry assistant.SkillRegistry
}

// NewListAvailableSkillsImpl creates a ListAvailableSkillsImpl.
func NewListAvailableSkillsImpl(skillRegistry assistant.SkillRegistry) *ListAvailableSkillsImpl {
	return &ListAvailableSkillsImpl{skillRegistry: skillRegistry}
}

// Query implements ListAvailableSkills.
func (uc ListAvailableSkillsImpl) Query(ctx context.Context) ([]assistant.SkillDefinition, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	skills, err := uc.skillRegistry.ListSkills(spanCtx)
	if telemetry.IsErrorRecorded(span, err) {
		return nil, err
	}

	return skills, nil
}
