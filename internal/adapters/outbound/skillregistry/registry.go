package skillregistry

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"slices"
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
	embeddedSkills []embeddedSkill
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
		embeddedSkills: embedded,
		encoder:        encoder,
		embeddingModel: embeddingModel,
		cfg:            normalized,
	}, nil
}

// ListRelevant returns only the top relevant skills for the given turn context.
func (r Registry) ListRelevant(ctx context.Context, query assistant.SkillQueryContext) []assistant.SkillDefinition {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	if r.encoder == nil || strings.TrimSpace(r.embeddingModel) == "" || len(r.embeddedSkills) == 0 {
		return nil
	}

	if forced := r.resolveForcedSkills(query.Messages); len(forced) > 0 {
		forcedNames := make([]string, 0, len(forced))
		for _, skill := range forced {
			forcedNames = append(forcedNames, skill.Name)
		}
		span.SetAttributes(attribute.StringSlice("skillregistry.forced_skill_names", forcedNames))
		span.SetAttributes(attribute.StringSlice("skillregistry.relevant_skill_names", forcedNames))
		return forced
	}

	currentInput, recentInputs := buildSelectionInputs(query.Messages, r.cfg.SelectionMaxChars, r.cfg.RecentInputsLimit)
	if currentInput == "" {
		return nil
	}

	hasPriorContext := recentInputs != "" || strings.TrimSpace(query.ConversationSummary) != ""
	contextMinScore := r.cfg.RelevantSkillsMinScore
	hasSubstantialInputs := hasSubstantialRecentInputs(recentInputs) || isSubstantialContextText(query.ConversationSummary)
	if hasPriorContext && hasSubstantialInputs {
		contextMinScore = max(0, contextMinScore-(r.cfg.LatestIntentOverrideDelta/2))
	}

	scored := r.rankSkills(spanCtx, currentInput, recentInputs, query.ConversationSummary, contextMinScore, true)
	latestInput := latestUserInput(query.Messages, r.cfg.SelectionMaxChars)
	if latestInput == "" {
		return nil
	}

	latestOnly := r.rankSkills(spanCtx, latestInput, "", "", 0, false)
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

// ListSkills returns all registered skills in stable priority order.
func (r Registry) ListSkills(ctx context.Context) ([]assistant.SkillDefinition, error) {
	_, span := telemetry.Start(ctx)
	defer span.End()

	return copySkillDefinitions(r.definitions), nil
}

// embedSkills converts the skill definitions into their vector representations for ranking.
func embedSkills(ctx context.Context, encoder semantic.Encoder, embeddingModel string, skills []assistant.SkillDefinition) ([]embeddedSkill, error) {
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

// hasSubstantialRecentInputs checks whether any recent user line is detailed
// enough to count as real context for score-threshold relaxation.
func hasSubstantialRecentInputs(recentInputs string) bool {
	return slices.ContainsFunc(strings.Split(recentInputs, "\n"), isSubstantialContextText)
}

// isSubstantialContextText classifies a message as meaningful context and
// excludes short acknowledgement-style replies.
func isSubstantialContextText(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false
	}
	if len([]rune(trimmed)) >= 20 {
		return true
	}
	return len(strings.Fields(trimmed)) >= 4
}
