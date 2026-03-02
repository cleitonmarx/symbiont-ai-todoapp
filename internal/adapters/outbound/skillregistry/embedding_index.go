package skillregistry

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// embedSkills builds the in-memory embedding index used during skill ranking.
func embedSkills(ctx context.Context, encoder domain.SemanticEncoder, embeddingModel string, skills []domain.AssistantSkillDefinition) ([]embeddedSkill, error) {
	embedded := make([]embeddedSkill, 0, len(skills))
	for _, skill := range skills {
		useVector, avoidVector, err := encoder.VectorizeSkillDefinition(ctx, embeddingModel, skill)
		if err != nil {
			return nil, fmt.Errorf("failed to vectorize skill %q: %w", skill.Name, err)
		}
		if len(useVector.Vector) == 0 {
			return nil, fmt.Errorf("empty use_when embedding for skill %q", skill.Name)
		}

		embedded = append(embedded, embeddedSkill{
			definition:  skill,
			useVector:   useVector.Vector,
			avoidVector: avoidVector.Vector,
		})
	}
	return embedded, nil
}
