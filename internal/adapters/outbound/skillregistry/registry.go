package skillregistry

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

// Registry owns the embedded skill index and resolves the most relevant skills
// for a conversation turn.
type Registry struct {
	definitions    []assistant.SkillDefinition
	embedded       []embeddedSkill
	encoder        semantic.Encoder
	embeddingModel string
	cfg            Config
}

//go:embed skills/*.md
var skillDirectory embed.FS

// NewSkillRegistryFromFS builds a skill registry by loading markdown files from the given filesystem.
func NewSkillRegistryFromFS(ctx context.Context, encoder semantic.Encoder, embeddingModel string, cfg Config) (Registry, error) {
	skills, err := LoadSkillsFromFS(skillDirectory)
	if err != nil {
		return Registry{}, err
	}
	return NewSkillRegistry(ctx, skills, encoder, embeddingModel, cfg)
}

// NewSkillRegistry builds an embedding-backed registry from pre-loaded skill definitions.
func NewSkillRegistry(ctx context.Context, skills []assistant.SkillDefinition, encoder semantic.Encoder, embeddingModel string, cfg Config) (Registry, error) {
	if encoder == nil {
		return Registry{}, errors.New("semantic encoder is required")
	}
	embeddingModel = strings.TrimSpace(embeddingModel)
	if embeddingModel == "" {
		return Registry{}, errors.New("embedding model is required")
	}

	normalized := normalizeConfig(cfg)
	definitions := copySkillDefinitions(skills)
	sort.Slice(definitions, func(i, j int) bool {
		if definitions[i].Priority == definitions[j].Priority {
			return definitions[i].Name < definitions[j].Name
		}
		return definitions[i].Priority > definitions[j].Priority
	})

	embedded, err := embedSkills(ctx, encoder, embeddingModel, definitions)
	if err != nil {
		return Registry{}, err
	}

	return Registry{
		definitions:    definitions,
		embedded:       embedded,
		encoder:        encoder,
		embeddingModel: embeddingModel,
		cfg:            normalized,
	}, nil
}

// ListRelevant returns only the top relevant skills for the given turn context.
func (r Registry) ListRelevant(ctx context.Context, query assistant.SkillQueryContext) []assistant.SkillDefinition {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	if r.encoder == nil || strings.TrimSpace(r.embeddingModel) == "" || len(r.embedded) == 0 {
		return nil
	}

	currentInput, recentInputs := buildSelectionInputs(query.Messages, r.cfg.SelectionMaxChars, r.cfg.RecentInputsLimit)
	if currentInput == "" {
		return nil
	}

	scored := r.rankSkills(spanCtx, currentInput, recentInputs, query.ConversationSummary, r.cfg.RelevantSkillsMinScore, true)
	latestInput := latestUserInput(query.Messages, r.cfg.SelectionMaxChars)
	if latestInput == "" {
		return nil
	}

	latestOnly := r.rankSkills(spanCtx, latestInput, "", "", 0, false)
	hasPriorContext := recentInputs != "" || strings.TrimSpace(query.ConversationSummary) != ""
	scored = r.chooseRanking(scored, latestOnly, hasPriorContext)
	if len(scored) == 0 {
		return nil
	}

	limit := min(len(scored), r.cfg.RelevantSkillsTopK)
	relevant := make([]assistant.SkillDefinition, 0, limit)
	relevantNames := make([]string, 0, limit)
	for i := range limit {
		relevant = append(relevant, scored[i].definition)
		relevantNames = append(relevantNames, scored[i].definition.Name)
	}

	if r.cfg.LogScores {
		for i, skill := range scored {
			fmt.Printf("  %d. %s (score: %.2f)\n", i+1, skill.definition.Name, skill.score)
		}
	}

	span.SetAttributes(attribute.StringSlice("skillregistry.relevant_skill_names", relevantNames))
	return relevant
}
