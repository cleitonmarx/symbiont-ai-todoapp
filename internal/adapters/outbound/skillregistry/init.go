package skillregistry

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitLocalSkillRegistry registers a local skill registry backed by static markdown files.
type InitLocalSkillRegistry struct {
	Encoder        semantic.Encoder `resolve:""`
	EmbeddingModel string           `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize builds the skill registry from embedded markdown files and registers it in the dependency container.
func (i InitLocalSkillRegistry) Initialize(ctx context.Context) (context.Context, error) {
	registry, err := NewSkillRegistryFromFS(ctx, i.Encoder, i.EmbeddingModel, Config{})
	if err != nil {
		return ctx, fmt.Errorf("failed to initialize skill registry: %w", err)
	}

	depend.Register[assistant.SkillRegistry](registry)
	return ctx, nil
}
