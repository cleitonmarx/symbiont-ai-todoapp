package skillregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		cfg    Config
		assert func(*testing.T, Config)
	}{
		{
			name: "defaults",
			cfg:  Config{},
			assert: func(t *testing.T, got Config) {
				assert.Equal(t, defaultRelevantSkillsTopK, got.RelevantSkillsTopK)
				assert.Equal(t, defaultRelevantSkillsMinScore, got.RelevantSkillsMinScore)
				assert.Equal(t, defaultAvoidPenaltyWeight, got.AvoidPenaltyWeight)
				assert.Equal(t, defaultAvoidBlockThreshold, got.AvoidBlockThreshold)
				assert.Equal(t, defaultStrongUseWhenScore, got.StrongUseWhenScore)
				assert.Equal(t, defaultCurrentInputWeight, got.CurrentInputWeight)
				assert.Equal(t, defaultRecentInputsWeight, got.RecentInputsWeight)
				assert.Equal(t, defaultSummaryWeight, got.SummaryWeight)
				assert.Equal(t, defaultRecentInputsLimit, got.RecentInputsLimit)
				assert.Equal(t, defaultSelectionMaxChars, got.SelectionMaxChars)
				assert.Equal(t, defaultLatestIntentOverrideDelta, got.LatestIntentOverrideDelta)
				assert.False(t, got.LogScores)
			},
		},
		{
			name: "explicit-and-negative-values",
			cfg: Config{
				RelevantSkillsTopK:        5,
				RelevantSkillsMinScore:    0.33,
				AvoidPenaltyWeight:        0.44,
				AvoidBlockThreshold:       0.55,
				StrongUseWhenScore:        0.66,
				CurrentInputWeight:        0.77,
				RecentInputsWeight:        -1,
				SummaryWeight:             -1,
				RecentInputsLimit:         9,
				SelectionMaxChars:         111,
				LogScores:                 true,
				LatestIntentOverrideDelta: 0.09,
			},
			assert: func(t *testing.T, got Config) {
				assert.Equal(t, 5, got.RelevantSkillsTopK)
				assert.Equal(t, 0.33, got.RelevantSkillsMinScore)
				assert.Equal(t, 0.44, got.AvoidPenaltyWeight)
				assert.Equal(t, 0.55, got.AvoidBlockThreshold)
				assert.Equal(t, 0.66, got.StrongUseWhenScore)
				assert.Equal(t, 0.77, got.CurrentInputWeight)
				assert.Equal(t, defaultRecentInputsWeight, got.RecentInputsWeight)
				assert.Equal(t, defaultSummaryWeight, got.SummaryWeight)
				assert.Equal(t, 9, got.RecentInputsLimit)
				assert.Equal(t, 111, got.SelectionMaxChars)
				assert.Equal(t, 0.09, got.LatestIntentOverrideDelta)
				assert.True(t, got.LogScores)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, normalizeConfig(tt.cfg))
		})
	}
}
