package skillregistry

import (
	"testing"
	"testing/fstest"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSkillRegistry_ValidatesDependencies(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		enc    semantic.Encoder
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
			enc:   semantic.NewMockEncoder(t),
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
			_, err := NewSkillRegistry(t.Context(), []assistant.SkillDefinition{{Name: "x", Content: "y"}}, tt.enc, tt.model, Config{})
			tt.assert(t, err)
		})
	}
}

func TestLocalRegistry_ListRelevant_EmbeddingRankingAndAvoid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		query    assistant.SkillQueryContext
		wantTop  string
		wantSize int
	}{
		"delete-intent-prefers-mutation-skill": {
			query: assistant.SkillQueryContext{
				Messages: []assistant.Message{{Role: assistant.ChatRole_User, Content: "please delete my groceries todos"}},
			},
			wantTop:  "todo-mutation-safety",
			wantSize: 2,
		},
		"chat-summary-avoids-mutation-skill": {
			query: assistant.SkillQueryContext{
				Messages: []assistant.Message{{Role: assistant.ChatRole_User, Content: "just chat summary"}},
			},
			wantTop:  "planning",
			wantSize: 2,
		},
		"empty-query-returns-none": {
			query:    assistant.SkillQueryContext{},
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

	registry, err := NewSkillRegistry(t.Context(), skills, encoder, "embed-model", Config{
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
			got := registry.ListRelevant(t.Context(), tt.query)
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

func TestLocalRegistry_ListRelevant_ForcedSkillDirectives(t *testing.T) {
	t.Parallel()

	skillsFS := fstest.MapFS{
		"planning.md": {Data: []byte(`---
name: planning
aliases: [plan]
use_when: user asks to plan
priority: 40
tags: [planning]
tools: [create_todos]
---
Build a simple plan.
`)},
		"mutation.md": {Data: []byte(`---
name: todo-mutation-safety
aliases: [update]
use_when: update delete complete todos
priority: 90
tags: [todos, mutation]
tools: [update_todos]
---
Use safe mutation flow.
`)},
	}

	skills, err := LoadSkillsFromFS(skillsFS)
	require.NoError(t, err)

	encoder := newSemanticEncoder(t, "embed-model", semanticEncoderParams{
		QueryVectors: map[string][]float64{
			"/unknown delete my todos": {1, 0},
		},
		SkillVectors: map[string]skillVector{
			"planning":             {Use: []float64{0, 1}},
			"todo-mutation-safety": {Use: []float64{1, 0}},
		},
	})

	registry, err := NewSkillRegistry(t.Context(), skills, encoder, "embed-model", Config{
		RelevantSkillsTopK:     2,
		RelevantSkillsMinScore: 0.10,
	})
	require.NoError(t, err)

	tests := map[string]struct {
		messages  []assistant.Message
		wantNames []string
	}{
		"forces-single-skill-from-directive": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "/planning create a travel plan"},
			},
			wantNames: []string{"planning"},
		},
		"forces-multiple-skills-preserving-order": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "/todo-mutation-safety /planning do this"},
			},
			wantNames: []string{"todo-mutation-safety", "planning"},
		},
		"forces-skill-using-alias": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "/plan create a trip plan"},
			},
			wantNames: []string{"planning"},
		},
		"falls-back-to-ranking-when-directive-unknown": {
			messages: []assistant.Message{
				{Role: assistant.ChatRole_User, Content: "/unknown delete my todos"},
			},
			wantNames: []string{"todo-mutation-safety"},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := registry.ListRelevant(t.Context(), assistant.SkillQueryContext{
				Messages: tt.messages,
			})

			require.NotEmpty(t, got)
			names := make([]string, 0, len(got))
			for _, skill := range got {
				names = append(names, skill.Name)
			}
			assert.Equal(t, tt.wantNames, names)
		})
	}
}
