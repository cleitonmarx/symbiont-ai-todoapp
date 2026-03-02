package skillregistry

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewSkillRegistry_ValidatesDependencies(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		enc    domain.SemanticEncoder
		model  string
		assert func(*testing.T, error)
	}{
		"missing-semantic-encoder": {
			model: "model",
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "semantic encoder is required")
			},
		},
		"missing-embedding-model": {
			enc:   domain.NewMockSemanticEncoder(t),
			model: "",
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "embedding model is required")
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := NewSkillRegistry(context.Background(), []domain.AssistantSkillDefinition{{Name: "x", Content: "y"}}, tt.enc, tt.model, Config{})
			tt.assert(t, err)
		})
	}
}

func TestLocalRegistry_ListRelevant_EmbeddingRankingAndAvoid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		query    domain.AssistantSkillQueryContext
		wantTop  string
		wantSize int
	}{
		"delete-intent-prefers-mutation-skill": {
			query: domain.AssistantSkillQueryContext{
				Messages: []domain.AssistantMessage{{Role: domain.ChatRole_User, Content: "please delete my groceries todos"}},
			},
			wantTop:  "todo-mutation-safety",
			wantSize: 2,
		},
		"chat-summary-avoids-mutation-skill": {
			query: domain.AssistantSkillQueryContext{
				Messages: []domain.AssistantMessage{{Role: domain.ChatRole_User, Content: "just chat summary"}},
			},
			wantTop:  "planning",
			wantSize: 2,
		},
		"empty-query-returns-none": {
			query:    domain.AssistantSkillQueryContext{},
			wantTop:  "",
			wantSize: 0,
		},
	}

	skillsFS := fstest.MapFS{
		"mutation.md": {Data: []byte(`---
name: todo-mutation-safety
use_when: update delete complete todos using UUIDs
avoid_when: only chat summary
priority: 90
tags: [todos, mutation, safety, uuid]
tools: [fetch_todos, update_todos, update_todos_due_date, delete_todos]
---
Never invent todo IDs. Use fetch_todos before update_todos.
`)},
		"planning.md": {Data: []byte(`---
name: planning
use_when: user asks to plan upcoming tasks
priority: 40
tags: [planning]
tools: [create_todos]
---
Build a simple plan in steps.
`)},
	}

	skills, err := LoadSkillsFromFS(skillsFS)
	require.NoError(t, err)

	encoder := newSemanticEncoder(t, "embed-model", semanticEncoderParams{
		QueryVectors: map[string][]float64{
			"please delete my groceries todos": {1, 0},
			"just chat summary":                {0, 1},
		},
		SkillVectors: map[string]skillVector{
			"todo-mutation-safety": {Use: []float64{1, 0}, Avoid: []float64{0, 1}},
			"planning":             {Use: []float64{0, 1}},
		},
	})

	registry, err := NewSkillRegistry(context.Background(), skills, encoder, "embed-model", Config{
		RelevantSkillsTopK:     2,
		RelevantSkillsMinScore: 0.10,
		AvoidPenaltyWeight:     0.80,
		AvoidBlockThreshold:    0.40,
		StrongUseWhenScore:     0.60,
	})
	require.NoError(t, err)

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := registry.ListRelevant(context.Background(), tt.query)
			if tt.wantTop == "" {
				assert.Empty(t, got)
				return
			}
			require.NotEmpty(t, got)
			assert.Equal(t, tt.wantTop, got[0].Name)
			assert.LessOrEqual(t, len(got), tt.wantSize)
		})
	}
}

func TestInitLocalSkillRegistry_Initialize(t *testing.T) {
	t.Parallel()

	enc := domain.NewMockSemanticEncoder(t)
	enc.EXPECT().
		VectorizeSkillDefinition(mock.Anything, "test-embedding-model", mock.Anything).
		Return(domain.EmbeddingVector{Vector: []float64{1, 0}}, domain.EmbeddingVector{Vector: []float64{0, 1}}, nil)

	i := InitLocalSkillRegistry{
		SemanticEncoder: enc,
		EmbeddingModel:  "test-embedding-model",
	}

	ctx, err := i.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	dep, err := depend.Resolve[domain.AssistantSkillRegistry]()
	assert.NoError(t, err)
	assert.NotNil(t, dep)

}
