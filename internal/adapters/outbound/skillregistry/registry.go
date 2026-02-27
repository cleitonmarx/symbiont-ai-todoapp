package skillregistry

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"go.opentelemetry.io/otel/attribute"
	"go.yaml.in/yaml/v3"
)

const (
	defaultRelevantSkillsTopK     = 3
	defaultRelevantSkillsMinScore = 0.20
	defaultAvoidPenaltyWeight     = 0.80
	defaultAvoidBlockThreshold    = 0.45
	defaultStrongUseWhenScore     = 0.55
	defaultCurrentInputWeight     = 0.65
	defaultRecentInputsWeight     = 0.25
	defaultSummaryWeight          = 0.10
	defaultRecentInputsLimit      = 3
	defaultSelectionMaxChars      = 400
)

// Config controls skill relevance ranking behavior.
type Config struct {
	RelevantSkillsTopK     int
	RelevantSkillsMinScore float64
	AvoidPenaltyWeight     float64
	AvoidBlockThreshold    float64
	StrongUseWhenScore     float64
	CurrentInputWeight     float64
	RecentInputsWeight     float64
	SummaryWeight          float64
	RecentInputsLimit      int
	SelectionMaxChars      int
}

// Registry stores static markdown skills and ranks relevance with embeddings.
type Registry struct {
	definitions         []domain.AssistantSkillDefinition
	embedded            []embeddedSkill
	encoder             domain.SemanticEncoder
	embeddingModel      string
	topK                int
	minScore            float64
	avoidPenaltyWeight  float64
	avoidBlockThreshold float64
	strongUseWhenScore  float64
	currentInputWeight  float64
	recentInputsWeight  float64
	summaryWeight       float64
	recentInputsLimit   int
	selectionMaxChars   int
}

// skillFrontMatter represents the expected YAML frontmatter structure in skill markdown files.
type skillFrontMatter struct {
	Name      string   `yaml:"name"`
	UseWhen   string   `yaml:"use_when"`
	AvoidWhen string   `yaml:"avoid_when"`
	Priority  int      `yaml:"priority"`
	Tags      []string `yaml:"tags"`
	Tools     []string `yaml:"tools"`
}

// embeddedSkill combines a skill definition with its pre-computed use and
// avoid vectors for efficient relevance scoring.
type embeddedSkill struct {
	definition  domain.AssistantSkillDefinition
	useVector   []float64
	avoidVector []float64
}

// scoredSkill pairs a skill definition with its calculated relevance score for a
// given user input, used for sorting and filtering relevant skills.
type scoredSkill struct {
	definition domain.AssistantSkillDefinition
	score      float64
}

type weightedQueryVector struct {
	weight float64
	vector []float64
}

//go:embed skills/*.md
var skillDirectory embed.FS

// NewSkillRegistryFromFS builds a skill registry by loading markdown files from the given filesystem.
func NewSkillRegistryFromFS(ctx context.Context, encoder domain.SemanticEncoder, embeddingModel string, cfg Config) (Registry, error) {
	skills, err := LoadSkillsFromFS(skillDirectory)
	if err != nil {
		return Registry{}, err
	}
	return NewSkillRegistry(ctx, skills, encoder, embeddingModel, cfg)
}

// NewSkillRegistry builds an embedding-backed registry from pre-loaded skill definitions.
func NewSkillRegistry(ctx context.Context, skills []domain.AssistantSkillDefinition, encoder domain.SemanticEncoder, embeddingModel string, cfg Config) (Registry, error) {
	if encoder == nil {
		return Registry{}, errors.New("semantic encoder is required")
	}
	embeddingModel = strings.TrimSpace(embeddingModel)
	if embeddingModel == "" {
		return Registry{}, errors.New("embedding model is required")
	}

	topK := cfg.RelevantSkillsTopK
	if topK <= 0 {
		topK = defaultRelevantSkillsTopK
	}

	minScore := cfg.RelevantSkillsMinScore
	if minScore <= 0 {
		minScore = defaultRelevantSkillsMinScore
	}

	avoidPenaltyWeight := cfg.AvoidPenaltyWeight
	if avoidPenaltyWeight <= 0 {
		avoidPenaltyWeight = defaultAvoidPenaltyWeight
	}

	avoidBlockThreshold := cfg.AvoidBlockThreshold
	if avoidBlockThreshold <= 0 {
		avoidBlockThreshold = defaultAvoidBlockThreshold
	}

	strongUseWhenScore := cfg.StrongUseWhenScore
	if strongUseWhenScore <= 0 {
		strongUseWhenScore = defaultStrongUseWhenScore
	}

	currentInputWeight := cfg.CurrentInputWeight
	if currentInputWeight <= 0 {
		currentInputWeight = defaultCurrentInputWeight
	}

	recentInputsWeight := cfg.RecentInputsWeight
	if recentInputsWeight < 0 {
		recentInputsWeight = 0
	}
	if recentInputsWeight == 0 {
		recentInputsWeight = defaultRecentInputsWeight
	}

	summaryWeight := cfg.SummaryWeight
	if summaryWeight < 0 {
		summaryWeight = 0
	}
	if summaryWeight == 0 {
		summaryWeight = defaultSummaryWeight
	}

	recentInputsLimit := cfg.RecentInputsLimit
	if recentInputsLimit <= 0 {
		recentInputsLimit = defaultRecentInputsLimit
	}

	selectionMaxChars := cfg.SelectionMaxChars
	if selectionMaxChars <= 0 {
		selectionMaxChars = defaultSelectionMaxChars
	}

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
		definitions:         definitions,
		embedded:            embedded,
		encoder:             encoder,
		embeddingModel:      embeddingModel,
		topK:                topK,
		minScore:            minScore,
		avoidPenaltyWeight:  avoidPenaltyWeight,
		avoidBlockThreshold: avoidBlockThreshold,
		strongUseWhenScore:  strongUseWhenScore,
		currentInputWeight:  currentInputWeight,
		recentInputsWeight:  recentInputsWeight,
		summaryWeight:       summaryWeight,
		recentInputsLimit:   recentInputsLimit,
		selectionMaxChars:   selectionMaxChars,
	}, nil
}

// ListRelevant returns only the top relevant skills for the given turn context.
func (r Registry) ListRelevant(ctx context.Context, query domain.AssistantSkillQueryContext) []domain.AssistantSkillDefinition {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	if r.encoder == nil || strings.TrimSpace(r.embeddingModel) == "" || len(r.embedded) == 0 {
		return nil
	}

	currentInput, recentInputs := buildSelectionInputs(query.Messages, r.selectionMaxChars, r.recentInputsLimit)
	if currentInput == "" {
		return nil
	}

	currentVector, err := r.encoder.VectorizeQuery(spanCtx, r.embeddingModel, currentInput)
	if err != nil || len(currentVector.Vector) == 0 {
		return nil
	}

	var recentVector []float64
	if recentInputs != "" {
		vec, err := r.encoder.VectorizeQuery(spanCtx, r.embeddingModel, recentInputs)
		if err == nil && len(vec.Vector) > 0 {
			recentVector = vec.Vector
		}
	}

	var summaryVector []float64
	summary := truncateToLastChars(strings.TrimSpace(query.ConversationSummary), r.selectionMaxChars)
	if summary != "" {
		vec, err := r.encoder.VectorizeQuery(spanCtx, r.embeddingModel, summary)
		if err == nil && len(vec.Vector) > 0 {
			summaryVector = vec.Vector
		}
	}

	queryVectors := []weightedQueryVector{
		{weight: r.currentInputWeight, vector: currentVector.Vector},
		{weight: r.recentInputsWeight, vector: recentVector},
		{weight: r.summaryWeight, vector: summaryVector},
	}

	scored := make([]scoredSkill, 0, len(r.embedded))
	for _, skill := range r.embedded {
		score, ok := r.scoreSkill(queryVectors, skill)
		if !ok || score < r.minScore {
			continue
		}
		scored = append(scored, scoredSkill{definition: skill.definition, score: score})
	}

	if len(scored) == 0 {
		return nil
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			if scored[i].definition.Priority == scored[j].definition.Priority {
				return scored[i].definition.Name < scored[j].definition.Name
			}
			return scored[i].definition.Priority > scored[j].definition.Priority
		}
		return scored[i].score > scored[j].score
	})

	limit := min(len(scored), r.topK)
	relevant := make([]domain.AssistantSkillDefinition, 0, limit)
	relevantNames := make([]string, 0, limit)
	for i := range limit {
		relevant = append(relevant, scored[i].definition)
		relevantNames = append(relevantNames, scored[i].definition.Name)
	}

	span.SetAttributes(
		attribute.StringSlice("skillregistry.relevant_skill_names", relevantNames),
	)

	return relevant
}

// scoreSkill calculates a relevance score for a skill based on weighted cosine similarity
// of current input, recent inputs, and optional summary vectors.
func (r Registry) scoreSkill(queryVectors []weightedQueryVector, skill embeddedSkill) (float64, bool) {
	useScore, ok := weightedSimilarity(queryVectors, skill.useVector)
	if !ok {
		return 0, false
	}

	avoidScore := 0.0
	if len(skill.avoidVector) > 0 {
		avoidScore, _ = weightedSimilarity(queryVectors, skill.avoidVector)
		if avoidScore >= r.avoidBlockThreshold && useScore < r.strongUseWhenScore {
			return 0, false
		}
	}

	score := useScore - (r.avoidPenaltyWeight * avoidScore) + priorityBoost(skill.definition.Priority)
	return score, true
}

func weightedSimilarity(queryVectors []weightedQueryVector, skillVector []float64) (float64, bool) {
	if len(skillVector) == 0 {
		return 0, false
	}

	weightedSum := 0.0
	totalWeight := 0.0
	for _, q := range queryVectors {
		if q.weight <= 0 || len(q.vector) == 0 {
			continue
		}
		sim, ok := common.CosineSimilarity(q.vector, skillVector)
		if !ok {
			continue
		}
		weightedSum += q.weight * sim
		totalWeight += q.weight
	}

	if totalWeight == 0 {
		return 0, false
	}

	return weightedSum / totalWeight, true
}

func buildSelectionInputs(messages []domain.AssistantMessage, maxChars int, recentLimit int) (string, string) {
	if len(messages) == 0 {
		return "", ""
	}

	currentIndex := -1
	currentInput := ""
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != domain.ChatRole_User {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		currentIndex = i
		currentInput = truncateToLastChars(content, maxChars)
		break
	}
	if currentIndex == -1 || currentInput == "" {
		return "", ""
	}

	if recentLimit <= 0 {
		return currentInput, ""
	}

	recent := make([]string, 0, recentLimit)
	for i := currentIndex - 1; i >= 0 && len(recent) < recentLimit; i-- {
		msg := messages[i]
		if msg.Role != domain.ChatRole_User {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		recent = append(recent, truncateToLastChars(content, maxChars))
	}

	if len(recent) == 0 {
		return currentInput, ""
	}

	// keep chronological order (oldest to newest) for a better context sentence.
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}

	return currentInput, strings.Join(recent, "\n")
}

func truncateToLastChars(input string, maxChars int) string {
	trimmed := strings.TrimSpace(input)
	if maxChars <= 0 {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) <= maxChars {
		return trimmed
	}

	return string(runes[len(runes)-maxChars:])
}

// embedSkills generates use and avoid vectors for each skill definition.
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

// priorityBoost calculates a small relevance boost based on skill priority to help break ties and promote important skills.
func priorityBoost(priority int) float64 {
	if priority <= 0 {
		return 0
	}
	return float64(priority) / 1000
}

// LoadSkillsFromFS loads and parses all markdown skill files from a filesystem root.
func LoadSkillsFromFS(skillsFS fs.FS) ([]domain.AssistantSkillDefinition, error) {
	if skillsFS == nil {
		return nil, errors.New("skills fs is nil")
	}

	skills := make([]domain.AssistantSkillDefinition, 0)
	err := fs.WalkDir(skillsFS, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		content, err := fs.ReadFile(skillsFS, path)
		if err != nil {
			return fmt.Errorf("failed to read skill file %q: %w", path, err)
		}

		skill, err := parseSkillMarkdown(path, content)
		if err != nil {
			return fmt.Errorf("failed to parse skill file %q: %w", path, err)
		}

		skills = append(skills, skill)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Priority == skills[j].Priority {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].Priority > skills[j].Priority
	})

	return skills, nil
}

// parseSkillMarkdown extracts skill definition data from a markdown file with YAML frontmatter.
func parseSkillMarkdown(path string, content []byte) (domain.AssistantSkillDefinition, error) {
	raw := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return domain.AssistantSkillDefinition{}, errors.New("missing YAML frontmatter opening delimiter")
	}

	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx == -1 {
		return domain.AssistantSkillDefinition{}, errors.New("missing YAML frontmatter closing delimiter")
	}

	metaRaw := strings.Join(lines[1:endIdx], "\n")
	var meta skillFrontMatter
	if err := yaml.Unmarshal([]byte(metaRaw), &meta); err != nil {
		return domain.AssistantSkillDefinition{}, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	name := strings.TrimSpace(meta.Name)
	if name == "" {
		return domain.AssistantSkillDefinition{}, errors.New("skill name is required")
	}

	body := strings.TrimSpace(strings.Join(lines[endIdx+1:], "\n"))
	if body == "" {
		return domain.AssistantSkillDefinition{}, errors.New("skill content is required")
	}

	return domain.AssistantSkillDefinition{
		Name:      name,
		UseWhen:   strings.TrimSpace(meta.UseWhen),
		AvoidWhen: strings.TrimSpace(meta.AvoidWhen),
		Priority:  max(0, meta.Priority),
		Tags:      sanitizeStringList(meta.Tags),
		Tools:     sanitizeStringList(meta.Tools),
		Content:   body,
		Source:    path,
	}, nil
}

// sanitizeStringList trims whitespace, removes empty values, and de-duplicates entries in a string slice.
func sanitizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	next := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		next = append(next, trimmed)
	}
	if len(next) == 0 {
		return nil
	}
	return next
}

// copySkillDefinitions creates deep copies of skill definitions to prevent accidental mutation of original data.
func copySkillDefinitions(skills []domain.AssistantSkillDefinition) []domain.AssistantSkillDefinition {
	if len(skills) == 0 {
		return nil
	}

	copied := make([]domain.AssistantSkillDefinition, 0, len(skills))
	for _, skill := range skills {
		tags := make([]string, len(skill.Tags))
		copy(tags, skill.Tags)
		skill.Tags = tags
		tools := make([]string, len(skill.Tools))
		copy(tools, skill.Tools)
		skill.Tools = tools
		copied = append(copied, skill)
	}
	return copied
}

// InitLocalSkillRegistry registers a local skill registry backed by static markdown files.
type InitLocalSkillRegistry struct {
	SemanticEncoder domain.SemanticEncoder `resolve:""`
	EmbeddingModel  string                 `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize loads skills and registers the domain skill registry.
func (i InitLocalSkillRegistry) Initialize(ctx context.Context) (context.Context, error) {
	registry, err := NewSkillRegistryFromFS(ctx, i.SemanticEncoder, i.EmbeddingModel, Config{
		RelevantSkillsTopK:     defaultRelevantSkillsTopK,
		RelevantSkillsMinScore: defaultRelevantSkillsMinScore,
		AvoidPenaltyWeight:     defaultAvoidPenaltyWeight,
		AvoidBlockThreshold:    defaultAvoidBlockThreshold,
		StrongUseWhenScore:     defaultStrongUseWhenScore,
		CurrentInputWeight:     defaultCurrentInputWeight,
		RecentInputsWeight:     defaultRecentInputsWeight,
		SummaryWeight:          defaultSummaryWeight,
		RecentInputsLimit:      defaultRecentInputsLimit,
		SelectionMaxChars:      defaultSelectionMaxChars,
	})
	if err != nil {
		return ctx, fmt.Errorf("failed to initialize skill registry: %w", err)
	}

	depend.Register[domain.AssistantSkillRegistry](registry)
	return ctx, nil
}
