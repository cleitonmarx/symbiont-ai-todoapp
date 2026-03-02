package skillregistry

import (
	"context"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_RankSkills(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		queryVectors semanticEncoderParams
		embedded     []embeddedSkill
		currentInput string
		recentInputs string
		summary      string
		minScore     float64
		wantNames    []string
	}{
		"empty-current-input-returns-none": {
			queryVectors: semanticEncoderParams{},
			embedded: []embeddedSkill{
				{definition: domain.AssistantSkillDefinition{Name: "todo-read-view"}, useVector: []float64{1, 0}},
			},
			currentInput: "   ",
			minScore:     0.10,
		},
		"ranks-using-all-query-vectors-and-applies-priority": {
			queryVectors: semanticEncoderParams{
				QueryVectors: map[string][]float64{
					"show matching todos":  {1, 0},
					"recent todo context":  {1, 0},
					"conversation summary": {0, 1},
				},
			},
			embedded: []embeddedSkill{
				{
					definition: domain.AssistantSkillDefinition{Name: "todo-read-view", Priority: 10},
					useVector:  []float64{1, 0},
				},
				{
					definition: domain.AssistantSkillDefinition{Name: "todo-summary", Priority: 80},
					useVector:  []float64{0.9, 0.4},
				},
			},
			currentInput: "show matching todos",
			recentInputs: "recent todo context",
			summary:      "conversation summary",
			minScore:     0.10,
			wantNames:    []string{"todo-summary", "todo-read-view"},
		},
		"ignores-recent-and-summary-errors": {
			queryVectors: semanticEncoderParams{
				QueryVectors: map[string][]float64{
					"mark it done": {1, 0},
				},
				QueryErrors: map[string]error{
					"recent context":      assert.AnError,
					"conversation memory": assert.AnError,
				},
			},
			embedded: []embeddedSkill{
				{
					definition: domain.AssistantSkillDefinition{Name: "todo-update"},
					useVector:  []float64{1, 0},
				},
			},
			currentInput: "mark it done",
			recentInputs: "recent context",
			summary:      "conversation memory",
			minScore:     0.10,
			wantNames:    []string{"todo-update"},
		},
		"filters-below-min-score": {
			queryVectors: semanticEncoderParams{
				QueryVectors: map[string][]float64{
					"search web": {1, 0},
				},
			},
			embedded: []embeddedSkill{
				{
					definition: domain.AssistantSkillDefinition{Name: "web-research"},
					useVector:  []float64{0.1, 1},
				},
			},
			currentInput: "search web",
			minScore:     0.50,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var encoder domain.SemanticEncoder
			if strings.TrimSpace(tt.currentInput) == "" {
				encoder = domain.NewMockSemanticEncoder(t)
			} else {
				encoder = newSemanticEncoder(t, "embed-model", tt.queryVectors)
			}
			registry := Registry{
				encoder:        encoder,
				embeddingModel: "embed-model",
				cfg:            normalizeConfig(Config{}),
				embedded:       tt.embedded,
			}

			got := registry.rankSkills(context.Background(), tt.currentInput, tt.recentInputs, tt.summary, tt.minScore, true)
			require.Len(t, got, len(tt.wantNames))
			for i, want := range tt.wantNames {
				assert.Equal(t, want, got[i].definition.Name)
			}
		})
	}
}

func TestRegistry_ChooseRanking(t *testing.T) {
	t.Parallel()

	registry := Registry{
		cfg: normalizeConfig(Config{
			RelevantSkillsTopK:        2,
			RelevantSkillsMinScore:    0.24,
			LatestIntentOverrideDelta: 0.05,
		}),
	}

	tests := map[string]struct {
		contextRanked   []scoredSkill
		latestOnly      []scoredSkill
		hasPriorContext bool
		wantNames       []string
	}{
		"returns-latest-when-context-empty-and-latest-above-min": {
			latestOnly: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "web-research"}, score: 0.30},
			},
			wantNames: []string{"web-research"},
		},
		"returns-none-when-context-empty-and-latest-below-min": {
			latestOnly: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "web-research"}, score: 0.20},
			},
		},
		"returns-context-when-latest-empty": {
			contextRanked: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "todo-update"}, score: 0.40},
			},
			wantNames: []string{"todo-update"},
		},
		"returns-context-when-top-skill-matches": {
			contextRanked: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "todo-read-view"}, score: 0.41},
			},
			latestOnly: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "todo-read-view"}, score: 0.60},
			},
			wantNames: []string{"todo-read-view"},
		},
		"overrides-when-latest-significantly-higher": {
			contextRanked: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "todo-goal-planner"}, score: 0.30},
			},
			latestOnly: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "web-research"}, score: 0.36},
				{definition: domain.AssistantSkillDefinition{Name: "todo-summary"}, score: 0.20},
			},
			wantNames: []string{"web-research", "todo-summary"},
		},
		"overrides-when-prior-context-and-latest-has-clear-lead": {
			contextRanked: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "todo-goal-planner"}, score: 0.32},
			},
			latestOnly: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "web-research"}, score: 0.22},
				{definition: domain.AssistantSkillDefinition{Name: "todo-summary"}, score: 0.18},
			},
			hasPriorContext: true,
			wantNames:       []string{"web-research", "todo-summary"},
		},
		"keeps-context-when-no-prior-context-for-soft-override": {
			contextRanked: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "todo-goal-planner"}, score: 0.32},
			},
			latestOnly: []scoredSkill{
				{definition: domain.AssistantSkillDefinition{Name: "web-research"}, score: 0.22},
				{definition: domain.AssistantSkillDefinition{Name: "todo-summary"}, score: 0.18},
			},
			wantNames: []string{"todo-goal-planner"},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := registry.chooseRanking(tt.contextRanked, tt.latestOnly, tt.hasPriorContext)
			require.Len(t, got, len(tt.wantNames))
			for i, want := range tt.wantNames {
				assert.Equal(t, want, got[i].definition.Name)
			}
		})
	}
}

func TestTrimRanked(t *testing.T) {
	t.Parallel()

	scored := []scoredSkill{
		{definition: domain.AssistantSkillDefinition{Name: "a"}, score: 0.9},
		{definition: domain.AssistantSkillDefinition{Name: "b"}, score: 0.8},
		{definition: domain.AssistantSkillDefinition{Name: "c"}, score: 0.7},
	}

	tests := map[string]struct {
		input     []scoredSkill
		topK      int
		wantNames []string
	}{
		"nil-when-empty": {},
		"returns-all-when-topk-zero": {
			input:     scored,
			topK:      0,
			wantNames: []string{"a", "b", "c"},
		},
		"returns-all-when-topk-exceeds-length": {
			input:     scored,
			topK:      5,
			wantNames: []string{"a", "b", "c"},
		},
		"trims-to-topk": {
			input:     scored,
			topK:      2,
			wantNames: []string{"a", "b"},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := trimRanked(tt.input, tt.topK)
			require.Len(t, got, len(tt.wantNames))
			for i, want := range tt.wantNames {
				assert.Equal(t, want, got[i].definition.Name)
			}
		})
	}
}

func TestRegistry_ScoreSkill(t *testing.T) {
	t.Parallel()

	registry := Registry{
		cfg: normalizeConfig(Config{
			AvoidPenaltyWeight:  0.70,
			AvoidBlockThreshold: 0.45,
			StrongUseWhenScore:  0.55,
		}),
	}

	tests := map[string]struct {
		queryVectors    []weightedQueryVector
		skill           embeddedSkill
		includePriority bool
		wantScore       float64
		wantOk          bool
	}{
		"returns-false-without-use-similarity": {
			queryVectors: []weightedQueryVector{{weight: 1, vector: nil}},
			skill:        embeddedSkill{definition: domain.AssistantSkillDefinition{Name: "x"}, useVector: []float64{1, 0}},
			wantOk:       false,
		},
		"blocks-when-avoid-is-strong-and-use-is-not-strong": {
			queryVectors: []weightedQueryVector{{weight: 1, vector: []float64{1, 0}}},
			skill: embeddedSkill{
				definition:  domain.AssistantSkillDefinition{Name: "x"},
				useVector:   []float64{0.4, 1},
				avoidVector: []float64{1, 0},
			},
			wantOk: false,
		},
		"applies-avoid-penalty-and-priority": {
			queryVectors: []weightedQueryVector{{weight: 1, vector: []float64{1, 0}}},
			skill: embeddedSkill{
				definition:  domain.AssistantSkillDefinition{Name: "x", Priority: 50},
				useVector:   []float64{1, 0},
				avoidVector: []float64{0.6, 0.8},
			},
			includePriority: true,
			wantScore:       1 - (0.70 * 0.6) + 0.05,
			wantOk:          true,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := registry.scoreSkill(tt.queryVectors, tt.skill, tt.includePriority)
			assert.Equal(t, tt.wantOk, ok)
			if tt.wantOk {
				assert.InDelta(t, tt.wantScore, got, 0.0001)
			}
		})
	}
}

func TestWeightedSimilarity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		queryVectors []weightedQueryVector
		skillVector  []float64
		want         float64
		wantOk       bool
	}{
		"returns-false-with-empty-skill-vector": {
			queryVectors: []weightedQueryVector{{weight: 1, vector: []float64{1, 0}}},
			wantOk:       false,
		},
		"returns-false-when-all-query-vectors-are-ignored": {
			queryVectors: []weightedQueryVector{
				{weight: 0, vector: []float64{1, 0}},
				{weight: 1, vector: nil},
			},
			skillVector: []float64{1, 0},
			wantOk:      false,
		},
		"computes-weighted-average": {
			queryVectors: []weightedQueryVector{
				{weight: 0.75, vector: []float64{1, 0}},
				{weight: 0.25, vector: []float64{0, 1}},
			},
			skillVector: []float64{1, 0},
			want:        0.75,
			wantOk:      true,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := weightedSimilarity(tt.queryVectors, tt.skillVector)
			assert.Equal(t, tt.wantOk, ok)
			if tt.wantOk {
				assert.InDelta(t, tt.want, got, 0.0001)
			}
		})
	}
}

func TestPriorityBoost(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		priority int
		want     float64
	}{
		"non-positive-priority-has-no-boost": {
			priority: 0,
			want:     0,
		},
		"positive-priority-scales-to-small-bonus": {
			priority: 75,
			want:     0.075,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, priorityBoost(tt.priority), 0.0001)
		})
	}
}
