package skillregistry

import (
	"testing"
	"testing/fstest"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSkillMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		assert  func(*testing.T, assistant.SkillDefinition, error)
	}{
		{
			name: "valid-markdown",
			content: `---
name: todo-mutation-safety
use_when: User asks to update/delete/complete todos and no explicit UUIDs are present.
avoid_when: User is only chatting or requesting read-only summaries.
priority: 90
embed_first_content_line: true
tags: [todos, mutation, safety, uuid]
tools: [fetch_todos, update_todos, update_todos_due_date, delete_todos]
---

Goal: execute todo mutations safely and with valid arguments.
`,
			assert: func(t *testing.T, got assistant.SkillDefinition, err error) {
				require.NoError(t, err)
				assert.Equal(t, "todo-mutation-safety", got.Name)
				assert.Equal(t, 90, got.Priority)
				assert.True(t, got.EmbedFirstContentLine)
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
			assert: func(t *testing.T, _ assistant.SkillDefinition, err error) {
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
			assert: func(t *testing.T, _ assistant.SkillDefinition, err error) {
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
			assert: func(t *testing.T, _ assistant.SkillDefinition, err error) {
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

func TestLoadSkillsFromFS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fs     fstest.MapFS
		assert func(*testing.T, []assistant.SkillDefinition, error)
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
			assert: func(t *testing.T, got []assistant.SkillDefinition, err error) {
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
			assert: func(t *testing.T, _ []assistant.SkillDefinition, err error) {
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
