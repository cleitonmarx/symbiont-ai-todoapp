package skillregistry

import (
	"context"
	"sort"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
)

// defaultRequiredLeadOverSecond is the fallback minimum score gap used when
// LatestIntentOverrideDelta is not configured.
const defaultRequiredLeadOverSecond = 0.02

// embeddedSkill stores a skill definition together with its precomputed
// retrieval vectors.
type embeddedSkill struct {
	definition  assistant.SkillDefinition
	useVector   []float64
	avoidVector []float64
}

// scoredSkill represents one ranked skill candidate.
type scoredSkill struct {
	definition assistant.SkillDefinition
	score      float64
}

// weightedQueryVector carries one query embedding and its contribution weight.
type weightedQueryVector struct {
	weight float64
	vector []float64
}

// rankSkills embeds the supplied query inputs and produces scored skill
// candidates ordered from most to least relevant.
func (r Registry) rankSkills(ctx context.Context, currentInput, recentInputs, summary string, minScore float64, includePriority bool) []scoredSkill {
	currentInput = truncateToLastChars(strings.TrimSpace(currentInput), r.cfg.SelectionMaxChars)
	if currentInput == "" {
		return nil
	}

	currentVector, err := r.encoder.VectorizeQuery(ctx, r.embeddingModel, currentInput)
	if err != nil || len(currentVector.Vector) == 0 {
		return nil
	}

	var recentVector []float64
	recentInputs = truncateToLastChars(recentInputs, r.cfg.SelectionMaxChars)
	if recentInputs != "" {
		vec, err := r.encoder.VectorizeQuery(ctx, r.embeddingModel, recentInputs)
		if err == nil && len(vec.Vector) > 0 {
			recentVector = vec.Vector
		}
	}

	var summaryVector []float64
	summary = truncateToLastChars(summary, r.cfg.SelectionMaxChars)
	if summary != "" {
		vec, err := r.encoder.VectorizeQuery(ctx, r.embeddingModel, summary)
		if err == nil && len(vec.Vector) > 0 {
			summaryVector = vec.Vector
		}
	}

	queryVectors := []weightedQueryVector{
		{weight: r.cfg.CurrentInputWeight, vector: currentVector.Vector},
		{weight: r.cfg.RecentInputsWeight, vector: recentVector},
		{weight: r.cfg.SummaryWeight, vector: summaryVector},
	}

	scored := make([]scoredSkill, 0, len(r.embeddedSkills))
	for _, skill := range r.embeddedSkills {
		score, ok := r.scoreSkill(queryVectors, skill, includePriority)
		if !ok || score < minScore {
			continue
		}
		scored = append(scored, scoredSkill{definition: skill.definition, score: score})
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

	return scored
}

// chooseRanking picks whether to trust conversation context or the latest
// user message when those two signals disagree.
func (r Registry) chooseRanking(contextRanked, latestOnly []scoredSkill, hasPriorContext bool) []scoredSkill {
	if len(contextRanked) == 0 {
		return r.chooseUsingLatestWhenNoContext(latestOnly, hasPriorContext)
	}

	if len(latestOnly) == 0 {
		return contextRanked
	}

	contextTop := contextRanked[0]
	latestTop := latestOnly[0]

	// If both strategies agree on the same top skill, keep context ranking to
	// preserve continuity and secondary candidates.
	if contextTop.definition.Name == latestTop.definition.Name {
		return contextRanked
	}

	if r.latestIsClearlyBetter(contextTop, latestTop) ||
		r.latestLooksLikeIntentChange(latestOnly, hasPriorContext) {
		return trimRanked(latestOnly, r.cfg.RelevantSkillsTopK)
	}

	return contextRanked
}

// chooseUsingLatestWhenNoContext handles turns where context ranking is empty.
// It accepts borderline scores only when prior context exists and the best
// latest candidate is clearly ahead of the runner-up.
func (r Registry) chooseUsingLatestWhenNoContext(latestOnly []scoredSkill, hasPriorContext bool) []scoredSkill {
	if len(latestOnly) == 0 {
		return nil
	}

	latestTop := latestOnly[0]
	if latestTop.score >= r.cfg.RelevantSkillsMinScore {
		return trimRanked(latestOnly, r.cfg.RelevantSkillsTopK)
	}

	secondBestScore := scoreOfSecondLatest(latestOnly)
	if !hasPriorContext {
		return nil
	}
	if latestTop.score < r.cfg.RelevantSkillsMinScore-r.cfg.LatestIntentOverrideDelta {
		return nil
	}
	if latestTop.score-secondBestScore < r.requiredLeadOverSecond() {
		return nil
	}

	return trimRanked(latestOnly, r.cfg.RelevantSkillsTopK)
}

// latestIsClearlyBetter returns true when the latest-message top skill is far
// enough above the context-based top skill.
func (r Registry) latestIsClearlyBetter(contextTop, latestTop scoredSkill) bool {
	return latestTop.score-contextTop.score >= r.cfg.LatestIntentOverrideDelta
}

// latestLooksLikeIntentChange handles softer pivots where latest top is close
// to threshold but still clearly better than the next latest candidate.
func (r Registry) latestLooksLikeIntentChange(latestOnly []scoredSkill, hasPriorContext bool) bool {
	if !hasPriorContext || len(latestOnly) == 0 {
		return false
	}

	latestTop := latestOnly[0]
	if latestTop.score < r.cfg.RelevantSkillsMinScore-r.cfg.LatestIntentOverrideDelta {
		return false
	}

	return latestTop.score-scoreOfSecondLatest(latestOnly) >= r.requiredLeadOverSecond()
}

// requiredLeadOverSecond defines how much better the top latest candidate must
// be than the second latest candidate.
func (r Registry) requiredLeadOverSecond() float64 {
	requiredLead := r.cfg.LatestIntentOverrideDelta / 2
	if requiredLead <= 0 {
		return defaultRequiredLeadOverSecond
	}
	return requiredLead
}

// scoreOfSecondLatest returns runner-up score in latest-only ranking.
func scoreOfSecondLatest(latestOnly []scoredSkill) float64 {
	if len(latestOnly) <= 1 {
		return 0
	}
	return latestOnly[1].score
}

// trimRanked applies the configured top-k cutoff to a scored skill list.
func trimRanked(scored []scoredSkill, topK int) []scoredSkill {
	if len(scored) == 0 {
		return nil
	}
	if topK <= 0 || len(scored) <= topK {
		return scored
	}
	return scored[:topK]
}

// scoreSkill computes the final ranking score for one embedded skill.
func (r Registry) scoreSkill(queryVectors []weightedQueryVector, skill embeddedSkill, includePriority bool) (float64, bool) {
	useScore, ok := weightedSimilarity(queryVectors, skill.useVector)
	if !ok {
		return 0, false
	}

	avoidScore := 0.0
	if len(skill.avoidVector) > 0 {
		avoidScore, _ = weightedSimilarity(queryVectors, skill.avoidVector)
		if avoidScore >= r.cfg.AvoidBlockThreshold && useScore < r.cfg.StrongUseWhenScore {
			return 0, false
		}
	}

	score := useScore - (r.cfg.AvoidPenaltyWeight * avoidScore)
	if includePriority {
		score += priorityBoost(skill.definition.Priority)
	}
	return score, true
}

// weightedSimilarity calculates a weighted cosine similarity over one or more
// query vectors against a skill vector.
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
		sim, ok := semantic.CosineSimilarity(q.vector, skillVector)
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

// priorityBoost converts a skill priority into a small ranking bonus.
func priorityBoost(priority int) float64 {
	if priority <= 0 {
		return 0
	}
	return float64(priority) / 1000
}

// resolveForcedSkills checks the latest user message for slash directives that explicitly select skills and returns those skills if found.
func (r Registry) resolveForcedSkills(messages []assistant.Message) []assistant.SkillDefinition {
	directiveNames := parseSelectedSkillDirectives(latestUserMessage(messages))
	if len(directiveNames) == 0 {
		return nil
	}

	availableByName := make(map[string]assistant.SkillDefinition, len(r.definitions))
	for _, definition := range r.definitions {
		canonical := strings.ToLower(strings.TrimSpace(definition.Name))
		if canonical == "" {
			continue
		}
		availableByName[canonical] = definition
	}
	for _, definition := range r.definitions {
		for _, alias := range definition.Aliases {
			normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
			if normalizedAlias == "" {
				continue
			}
			if _, exists := availableByName[normalizedAlias]; exists {
				continue
			}
			availableByName[normalizedAlias] = definition
		}
	}

	forced := make([]assistant.SkillDefinition, 0, len(directiveNames))
	for _, name := range directiveNames {
		definition, ok := availableByName[name]
		if !ok {
			continue
		}
		forced = append(forced, definition)
	}

	if len(forced) == 0 {
		return nil
	}

	if r.cfg.RelevantSkillsTopK <= 0 || len(forced) <= r.cfg.RelevantSkillsTopK {
		return forced
	}
	return forced[:r.cfg.RelevantSkillsTopK]
}
