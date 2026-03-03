package mcp

import (
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToolOverrideDefinitions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content string
		assert  func(*testing.T, map[string]domain.AssistantActionDefinition, error)
	}{
		"valid-yaml": {
			content: `
tools:
  - name: search
    description: Search docs
    status_message: Searching docs...
    input:
      type: object
      fields:
        query:
          type: string
          description: Query
          required: true
  - name: "   "
    description: ignored
`,
			assert: func(t *testing.T, got map[string]domain.AssistantActionDefinition, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				require.Contains(t, got, "search")
				assert.Equal(t, "Search docs", got["search"].Description)
				assert.Equal(t, "string", got["search"].Input.Fields["query"].Type)
			},
		},
		"valid-yaml-with-approvals-override": {
			content: `
tools:
  - name: delete_todos
    approvals:
      required: true
      title: Confirm delete
      description: Destructive action.
      preview_fields:
        - todos[].title
        - todos[].id
      timeout: 45s
`,
			assert: func(t *testing.T, got map[string]domain.AssistantActionDefinition, err error) {
				require.NoError(t, err)
				require.Contains(t, got, "delete_todos")
				assert.True(t, got["delete_todos"].Approval.Required)
				assert.Equal(t, "Confirm delete", got["delete_todos"].Approval.Title)
				assert.Equal(t, []string{"todos[].title", "todos[].id"}, got["delete_todos"].Approval.PreviewFields)
				assert.Equal(t, 45*time.Second, got["delete_todos"].Approval.Timeout)
			},
		},
		"invalid-yaml-with-approval-timeout-format": {
			content: `
tools:
  - name: delete_todos
    approvals:
      timeout: 45
`,
			assert: func(t *testing.T, _ map[string]domain.AssistantActionDefinition, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid approval timeout")
			},
		},
		"invalid-yaml": {
			content: "tools: [",
			assert: func(t *testing.T, _ map[string]domain.AssistantActionDefinition, err error) {
				require.Error(t, err)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := parseToolOverrideDefinitions([]byte(tt.content))
			tt.assert(t, got, err)
		})
	}
}

func TestParseToolOverrideStatusMessages(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content string
		assert  func(*testing.T, map[string]string, error)
	}{
		"valid-yaml": {
			content: `
tools:
  - name: search
    status_message: Searching docs...
  - name: fetch_content
    status_message: "  "
`,
			assert: func(t *testing.T, got map[string]string, err error) {
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, "Searching docs...", got["search"])
			},
		},
		"invalid-yaml": {
			content: "tools: [",
			assert: func(t *testing.T, _ map[string]string, err error) {
				require.Error(t, err)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := parseToolOverrideStatusMessages([]byte(tt.content))
			tt.assert(t, got, err)
		})
	}
}

func TestApprovalOverrideHelpers(t *testing.T) {
	t.Parallel()

	t.Run("sanitize-preview-fields", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			input []string
			want  []string
		}{
			"empty":              {input: nil, want: nil},
			"trims-and-compacts": {input: []string{" todos[].title ", "", "todos[].title", "todos[].id"}, want: []string{"todos[].title", "todos[].id"}},
		}

		for name, tt := range tests {
			tt := tt
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.want, sanitizePreviewFields(tt.input))
			})
		}
	})

	t.Run("parse-approval-timeout", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			input   string
			want    time.Duration
			wantErr string
		}{
			"empty":   {input: "", want: 0},
			"valid":   {input: "45s", want: 45 * time.Second},
			"invalid": {input: "45", wantErr: "invalid approval timeout"},
		}

		for name, tt := range tests {
			tt := tt
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				got, err := parseApprovalTimeout(tt.input)
				if tt.wantErr != "" {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.wantErr)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("approval-config-to-domain", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			input   assistantActionApprovalConfig
			want    domain.AssistantActionApproval
			wantErr string
		}{
			"valid": {
				input: assistantActionApprovalConfig{Required: true, Title: " Confirm ", Description: " Desc ", PreviewFields: []string{" todos[].title "}, Timeout: "30s"},
				want:  domain.AssistantActionApproval{Required: true, Title: "Confirm", Description: "Desc", PreviewFields: []string{"todos[].title"}, Timeout: 30 * time.Second},
			},
			"invalid-timeout": {
				input:   assistantActionApprovalConfig{Timeout: "30"},
				wantErr: "invalid approval timeout",
			},
		}

		for name, tt := range tests {
			tt := tt
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.input.toDomain()
				if tt.wantErr != "" {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.wantErr)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})
}
