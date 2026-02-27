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

func TestParseSkillMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		assert  func(*testing.T, domain.AssistantSkillDefinition, error)
	}{
		{
			name: "valid-markdown",
			content: `---
name: todo-mutation-safety
use_when: User asks to update/delete/complete todos and no explicit UUIDs are present.
avoid_when: User is only chatting or requesting read-only summaries.
priority: 90
tags: [todos, mutation, safety, uuid]
tools: [fetch_todos, update_todos, update_todos_due_date, delete_todos]
---

Goal: execute todo mutations safely and with valid arguments.
`,
			assert: func(t *testing.T, got domain.AssistantSkillDefinition, err error) {
				require.NoError(t, err)
				assert.Equal(t, "todo-mutation-safety", got.Name)
				assert.Equal(t, 90, got.Priority)
				assert.Equal(t, []string{"todos", "mutation", "safety", "uuid"}, got.Tags)
				assert.Equal(t, []string{"fetch_todos", "update_todos", "update_todos_due_date", "delete_todos"}, got.Tools)
				assert.Equal(t, "Goal: execute todo mutations safely and with valid arguments.", got.Content)
				assert.Equal(t, "skills/mutation.md", got.Source)
			},
		},
		{
			name: "missing-frontmatter",
			content: `name: todo-mutation-safety
body only
`,
			assert: func(t *testing.T, _ domain.AssistantSkillDefinition, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "missing YAML frontmatter opening delimiter")
			},
		},
		{
			name: "missing-name",
			content: `---
use_when: test
---
body
`,
			assert: func(t *testing.T, _ domain.AssistantSkillDefinition, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "skill name is required")
			},
		},
		{
			name: "missing-body",
			content: `---
name: noop
---
`,
			assert: func(t *testing.T, _ domain.AssistantSkillDefinition, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "skill content is required")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseSkillMarkdown("skills/mutation.md", []byte(tt.content))
			tt.assert(t, got, err)
		})
	}
}

func TestLoadSkillsFromFS_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fs     fstest.MapFS
		assert func(*testing.T, []domain.AssistantSkillDefinition, error)
	}{
		{
			name: "loads-markdown-and-sorts-by-priority",
			fs: fstest.MapFS{
				"a.md": {Data: []byte(`---
name: low-priority
priority: 10
---
Low skill.
`)},
				"nested/b.md": {Data: []byte(`---
name: high-priority
priority: 90
tags: [todos]
tools: [fetch_todos]
---
High skill.
`)},
				"ignore.txt": {Data: []byte("ignored")},
			},
			assert: func(t *testing.T, got []domain.AssistantSkillDefinition, err error) {
				require.NoError(t, err)
				require.Len(t, got, 2)
				assert.Equal(t, "high-priority", got[0].Name)
				assert.Equal(t, "low-priority", got[1].Name)
				assert.Equal(t, "nested/b.md", got[0].Source)
				assert.Equal(t, []string{"fetch_todos"}, got[0].Tools)
			},
		},
		{
			name: "fails-on-invalid-markdown",
			fs: fstest.MapFS{
				"bad.md": {Data: []byte("no frontmatter")},
			},
			assert: func(t *testing.T, _ []domain.AssistantSkillDefinition, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to parse skill file")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := LoadSkillsFromFS(tt.fs)
			tt.assert(t, got, err)
		})
	}
}

func TestNewSkillRegistry_ValidatesDependencies(t *testing.T) {
	t.Parallel()

	_, err := NewSkillRegistry(context.Background(), []domain.AssistantSkillDefinition{{Name: "x", Content: "y"}}, nil, "model", Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "semantic encoder is required")

	_, err = NewSkillRegistry(context.Background(), []domain.AssistantSkillDefinition{{Name: "x", Content: "y"}}, domain.NewMockSemanticEncoder(t), "", Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embedding model is required")
}

func TestLocalRegistry_ListRelevant_EmbeddingRankingAndAvoid(t *testing.T) {
	t.Parallel()

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
			"todo-mutation-safety": {
				Use:   []float64{1, 0},
				Avoid: []float64{0, 1},
			},
			"planning": {
				Use: []float64{0, 1},
			},
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

	relevant := registry.ListRelevant(context.Background(), domain.AssistantSkillQueryContext{
		Messages: []domain.AssistantMessage{
			{Role: domain.ChatRole_User, Content: "please delete my groceries todos"},
		},
	})
	require.NotEmpty(t, relevant)
	assert.Equal(t, "todo-mutation-safety", relevant[0].Name)
	assert.LessOrEqual(t, len(relevant), 2)

	chatRelevant := registry.ListRelevant(context.Background(), domain.AssistantSkillQueryContext{
		Messages: []domain.AssistantMessage{
			{Role: domain.ChatRole_User, Content: "just chat summary"},
		},
	})
	require.NotEmpty(t, chatRelevant)
	assert.Equal(t, "planning", chatRelevant[0].Name)

	none := registry.ListRelevant(context.Background(), domain.AssistantSkillQueryContext{})
	assert.Empty(t, none)
}

func TestBuildSelectionInputs_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messages    []domain.AssistantMessage
		maxChars    int
		recentLimit int
		wantCurrent string
		wantRecent  string
	}{
		{
			name: "current-and-recent-user-inputs",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "plan trip to tokyo"},
				{Role: domain.ChatRole_Assistant, Content: "What dates?"},
				{Role: domain.ChatRole_User, Content: "april 5 to 18"},
			},
			maxChars:    400,
			recentLimit: 3,
			wantCurrent: "april 5 to 18",
			wantRecent:  "plan trip to tokyo",
		},
		{
			name: "respects-recent-limit",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_User, Content: "u1"},
				{Role: domain.ChatRole_User, Content: "u2"},
				{Role: domain.ChatRole_User, Content: "u3"},
				{Role: domain.ChatRole_User, Content: "u4"},
			},
			maxChars:    400,
			recentLimit: 2,
			wantCurrent: "u4",
			wantRecent:  "u2\nu3",
		},
		{
			name: "returns-empty-without-user-message",
			messages: []domain.AssistantMessage{
				{Role: domain.ChatRole_System, Content: "system"},
			},
			maxChars:    400,
			recentLimit: 3,
			wantCurrent: "",
			wantRecent:  "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotCurrent, gotRecent := buildSelectionInputs(tt.messages, tt.maxChars, tt.recentLimit)
			assert.Equal(t, tt.wantCurrent, gotCurrent)
			assert.Equal(t, tt.wantRecent, gotRecent)
		})
	}
}

func TestInitLocalSkillRegistry_Initialize(t *testing.T) {
	enc := domain.NewMockSemanticEncoder(t)

	enc.EXPECT().VectorizeSkillDefinition(mock.Anything, "test-embedding-model", mock.Anything).
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

type semanticEncoderParams struct {
	QueryVectors map[string][]float64
	QueryErrors  map[string]error
	SkillVectors map[string]skillVector
	SkillErrors  map[string]error
}

type skillVector struct {
	Use   []float64
	Avoid []float64
}

func newSemanticEncoder(t *testing.T, model string, params semanticEncoderParams) *domain.MockSemanticEncoder {
	t.Helper()

	enc := domain.NewMockSemanticEncoder(t)
	for query, vec := range params.QueryVectors {
		enc.EXPECT().
			VectorizeQuery(mock.Anything, model, query).
			Return(domain.EmbeddingVector{Vector: vec}, nil).
			Once()
	}
	for query, err := range params.QueryErrors {
		enc.EXPECT().
			VectorizeQuery(mock.Anything, model, query).
			Return(domain.EmbeddingVector{}, err).
			Once()
	}

	for name, vector := range params.SkillVectors {
		skillName := name
		enc.EXPECT().
			VectorizeSkillDefinition(
				mock.Anything,
				model,
				mock.MatchedBy(func(skill domain.AssistantSkillDefinition) bool { return skill.Name == skillName }),
			).
			Return(
				domain.EmbeddingVector{Vector: vector.Use},
				domain.EmbeddingVector{Vector: vector.Avoid},
				nil,
			).
			Once()
	}
	for name, err := range params.SkillErrors {
		skillName := name
		enc.EXPECT().
			VectorizeSkillDefinition(
				mock.Anything,
				model,
				mock.MatchedBy(func(skill domain.AssistantSkillDefinition) bool { return skill.Name == skillName }),
			).
			Return(domain.EmbeddingVector{}, domain.EmbeddingVector{}, err).
			Once()
	}

	return enc
}
