package skillregistry

import (
	"context"
	"sort"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// embeddedSkill stores a skill definition together with its precomputed
// retrieval vectors.
type embeddedSkill struct {
	definition  domain.AssistantSkillDefinition
	useVector   []float64
	avoidVector []float64
}

// scoredSkill represents one ranked skill candidate.
type scoredSkill struct {
	definition domain.AssistantSkillDefinition
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
	recentInputs = truncateToLastChars(strings.TrimSpace(recentInputs), r.cfg.SelectionMaxChars)
	if recentInputs != "" {
		vec, err := r.encoder.VectorizeQuery(ctx, r.embeddingModel, recentInputs)
		if err == nil && len(vec.Vector) > 0 {
			recentVector = vec.Vector
		}
	}

	var summaryVector []float64
	summary = truncateToLastChars(strings.TrimSpace(summary), r.cfg.SelectionMaxChars)
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

	scored := make([]scoredSkill, 0, len(r.embedded))
	for _, skill := range r.embedded {
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

// chooseRanking arbitrates between context-aware and latest-message-only
// rankings so explicit intent changes can override stale context.
func (r Registry) chooseRanking(contextRanked, latestOnly []scoredSkill, hasPriorContext bool) []scoredSkill {
	if len(contextRanked) == 0 {
		if len(latestOnly) == 0 || latestOnly[0].score < r.cfg.RelevantSkillsMinScore {
			return nil
		}
		return trimRanked(latestOnly, r.cfg.RelevantSkillsTopK)
	}
	if len(latestOnly) == 0 {
		return contextRanked
	}

	contextTop := contextRanked[0]
	latestTop := latestOnly[0]
	if contextTop.definition.Name == latestTop.definition.Name {
		return contextRanked
	}

	if latestTop.score-contextTop.score >= r.cfg.LatestIntentOverrideDelta {
		return trimRanked(latestOnly, r.cfg.RelevantSkillsTopK)
	}

	latestLeadDelta := r.cfg.LatestIntentOverrideDelta / 2
	if latestLeadDelta <= 0 {
		latestLeadDelta = 0.02
	}
	latestSecondScore := 0.0
	if len(latestOnly) > 1 {
		latestSecondScore = latestOnly[1].score
	}
	if hasPriorContext && latestTop.score >= r.cfg.RelevantSkillsMinScore-r.cfg.LatestIntentOverrideDelta && latestTop.score-latestSecondScore >= latestLeadDelta {
		return trimRanked(latestOnly, r.cfg.RelevantSkillsTopK)
	}

	return contextRanked
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

// priorityBoost converts a skill priority into a small ranking bonus.
func priorityBoost(priority int) float64 {
	if priority <= 0 {
		return 0
	}
	return float64(priority) / 1000
}
