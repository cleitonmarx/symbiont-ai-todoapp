package skillregistry

const (
	defaultRelevantSkillsTopK        = 2
	defaultRelevantSkillsMinScore    = 0.23
	defaultAvoidPenaltyWeight        = 0.70
	defaultAvoidBlockThreshold       = 0.45
	defaultStrongUseWhenScore        = 0.55
	defaultCurrentInputWeight        = 0.70
	defaultRecentInputsWeight        = 0.25
	defaultSummaryWeight             = 0.05
	defaultRecentInputsLimit         = 4
	defaultSelectionMaxChars         = 400
	defaultLatestIntentOverrideDelta = 0.07
)

// Config configures how the registry ranks and filters relevant skills.
type Config struct {
	// RelevantSkillsTopK controls how many of the top-ranked skills are returned for selection.
	RelevantSkillsTopK int
	// RelevantSkillsMinScore sets a minimum score threshold for skills to be considered relevant.
	RelevantSkillsMinScore float64
	// AvoidPenaltyWeight determines how strongly the avoid_when similarity penalizes a skill's final score.
	AvoidPenaltyWeight float64
	// AvoidBlockThreshold blocks a skill when avoid_when is very similar and use_when is not strong enough.
	AvoidBlockThreshold float64
	// StrongUseWhenScore is the use_when score required to bypass avoid_when blocking.
	StrongUseWhenScore float64
	// CurrentInputWeight gives the highest weight to the latest user request.
	CurrentInputWeight float64
	// RecentInputsWeight weights recent user inputs used for continuity.
	RecentInputsWeight float64
	// SummaryWeight weights conversation summary context during ranking.
	SummaryWeight float64
	// RecentInputsLimit controls how many recent user inputs are included in ranking context.
	RecentInputsLimit int
	// SelectionMaxChars keeps only trailing runes up to this size before vectorization.
	SelectionMaxChars int
	// LogScores enables logging of skill scores for debugging purposes.
	LogScores bool
	// LatestIntentOverrideDelta controls how far latest-intent scores must separate
	// before overriding context-based ranking.
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
