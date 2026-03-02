package skillregistry

const (
	// defaultRelevantSkillsTopK limits how many skills are returned after ranking.
	defaultRelevantSkillsTopK        = 3
	// defaultRelevantSkillsMinScore is the minimum score a skill must reach to be kept.
	defaultRelevantSkillsMinScore    = 0.24
	// defaultAvoidPenaltyWeight scales how much avoid_when similarity reduces the final score.
	defaultAvoidPenaltyWeight        = 0.70
	// defaultAvoidBlockThreshold blocks a skill entirely when avoid_when similarity is too strong.
	defaultAvoidBlockThreshold       = 0.45
	// defaultStrongUseWhenScore marks a skill as strongly relevant for scoring heuristics.
	defaultStrongUseWhenScore        = 0.55
	// defaultCurrentInputWeight gives the highest weight to the latest user request.
	defaultCurrentInputWeight        = 0.50
	// defaultRecentInputsWeight weights recent user inputs used for continuity.
	defaultRecentInputsWeight        = 0.40
	// defaultSummaryWeight weights conversation summary context during ranking.
	defaultSummaryWeight             = 0.10
	// defaultRecentInputsLimit controls how many recent user inputs are included in ranking context.
	defaultRecentInputsLimit         = 4
	// defaultSelectionMaxChars caps query text length before vectorization.
	defaultSelectionMaxChars         = 400
	// defaultLatestIntentOverrideDelta is the minimum score gap for latest-message intent override.
	defaultLatestIntentOverrideDelta = 0.05
)

// Config configures how the registry ranks and filters relevant skills.
type Config struct {
	RelevantSkillsTopK        int
	RelevantSkillsMinScore    float64
	AvoidPenaltyWeight        float64
	AvoidBlockThreshold       float64
	StrongUseWhenScore        float64
	CurrentInputWeight        float64
	RecentInputsWeight        float64
	SummaryWeight             float64
	RecentInputsLimit         int
	SelectionMaxChars         int
	LogScores                 bool
	LatestIntentOverrideDelta float64
}

// normalizeConfig applies default values and clamps invalid inputs so ranking
// code can operate on a complete, stable Config.
func normalizeConfig(cfg Config) Config {
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

	latestIntentOverrideDelta := cfg.LatestIntentOverrideDelta
	if latestIntentOverrideDelta <= 0 {
		latestIntentOverrideDelta = defaultLatestIntentOverrideDelta
	}

	return Config{
		RelevantSkillsTopK:        topK,
		RelevantSkillsMinScore:    minScore,
		AvoidPenaltyWeight:        avoidPenaltyWeight,
		AvoidBlockThreshold:       avoidBlockThreshold,
		StrongUseWhenScore:        strongUseWhenScore,
		CurrentInputWeight:        currentInputWeight,
		RecentInputsWeight:        recentInputsWeight,
		SummaryWeight:             summaryWeight,
		RecentInputsLimit:         recentInputsLimit,
		SelectionMaxChars:         selectionMaxChars,
		LogScores:                 cfg.LogScores,
		LatestIntentOverrideDelta: latestIntentOverrideDelta,
	}
}
