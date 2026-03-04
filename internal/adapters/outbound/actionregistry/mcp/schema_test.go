package mcp

import (
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolToActionDefinition(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tool   *mcp.Tool
		assert func(*testing.T, domain.AssistantActionDefinition)
	}{
		"nil-tool": {
			tool: nil,
			assert: func(t *testing.T, got domain.AssistantActionDefinition) {
				assert.Equal(t, domain.AssistantActionDefinition{}, got)
			},
		},
		"uses-description": {
			tool: &mcp.Tool{Name: "search", Description: " Search docs ", Title: "Ignored", InputSchema: map[string]any{"type": "object"}},
			assert: func(t *testing.T, got domain.AssistantActionDefinition) {
				assert.Equal(t, "search", got.Name)
				assert.Equal(t, "Search docs", got.Description)
				assert.Equal(t, "object", got.Input.Type)
			},
		},
		"falls-back-to-title": {
			tool: &mcp.Tool{Name: "search", Title: " Search docs ", InputSchema: map[string]any{"type": "object"}},
			assert: func(t *testing.T, got domain.AssistantActionDefinition) {
				assert.Equal(t, "Search docs", got.Description)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, toolToActionDefinition(tt.tool))
		})
	}
}

func TestSchemaToInput(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		schema any
		assert func(*testing.T, domain.AssistantActionInput)
	}{
		"nil-schema": {
			schema: nil,
			assert: func(t *testing.T, got domain.AssistantActionInput) {
				assert.Equal(t, "object", got.Type)
				assert.Empty(t, got.Fields)
			},
		},
		"simple-properties-required": {
			schema: map[string]any{"type": "object", "properties": map[string]any{"title": map[string]any{"type": "string", "description": "Task title"}}, "required": []any{"title"}},
			assert: func(t *testing.T, got domain.AssistantActionInput) {
				require.Contains(t, got.Fields, "title")
				assert.Equal(t, "string", got.Fields["title"].Type)
				assert.True(t, got.Fields["title"].Required)
			},
		},
		"nested-array-object-with-format-and-enum": {
			schema: map[string]any{"type": "object", "properties": map[string]any{"todos": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string", "description": "Todo id"}, "status": map[string]any{"type": "string", "enum": []any{"OPEN", "DONE"}, "format": "enum"}}, "required": []any{"id"}}}}, "required": []any{"todos"}},
			assert: func(t *testing.T, got domain.AssistantActionInput) {
				field := got.Fields["todos"]
				assert.Equal(t, "array", field.Type)
				assert.True(t, field.Required)
				require.NotNil(t, field.Items)
				assert.True(t, field.Items.Fields["id"].Required)
				assert.Equal(t, []any{"OPEN", "DONE"}, field.Items.Fields["status"].Enum)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, schemaToInput(tt.schema))
		})
	}
}

func TestSchemaHelpers(t *testing.T) {
	t.Parallel()

	t.Run("schema-field-type", func(t *testing.T) {
		t.Parallel()
		tests := map[string]struct {
			input map[string]any
			want  string
		}{
			"direct":   {input: map[string]any{"type": "string"}, want: "string"},
			"compound": {input: map[string]any{"oneOf": []any{map[string]any{"type": "integer"}, map[string]any{"type": "null"}}}, want: "integer|null"},
			"empty":    {input: map[string]any{}, want: ""},
		}
		for name, tt := range tests {
			tt := tt
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.want, schemaFieldType(tt.input))
			})
		}
	})

	t.Run("parse-type-value", func(t *testing.T) {
		t.Parallel()
		tests := map[string]struct {
			input any
			want  string
		}{
			"string": {input: "string", want: "string"},
			"slice":  {input: []any{"null", "integer", "integer"}, want: "integer|null"},
			"other":  {input: 1, want: ""},
		}
		for name, tt := range tests {
			tt := tt
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.want, parseTypeValue(tt.input))
			})
		}
	})

	t.Run("required-set", func(t *testing.T) {
		t.Parallel()
		tests := map[string]struct {
			input any
			want  map[string]bool
		}{
			"valid":   {input: []any{"title", "status"}, want: map[string]bool{"title": true, "status": true}},
			"invalid": {input: "title", want: map[string]bool{}},
		}
		for name, tt := range tests {
			tt := tt
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.want, requiredSet(tt.input))
			})
		}
	})

	t.Run("approval-override", func(t *testing.T) {
		t.Parallel()
		tests := map[string]struct {
			input domain.AssistantActionApproval
			want  bool
		}{
			"empty":   {input: domain.AssistantActionApproval{}, want: false},
			"present": {input: domain.AssistantActionApproval{Timeout: time.Second}, want: true},
		}
		for name, tt := range tests {
			tt := tt
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.want, hasApprovalOverride(tt.input))
			})
		}
	})
}

func TestMergeAssistantActionDefinition(t *testing.T) {
	t.Parallel()

	base := domain.AssistantActionDefinition{Name: "search", Description: "base description", Input: domain.AssistantActionInput{Type: "object", Fields: map[string]domain.AssistantActionField{"query": {Type: "string", Description: "q", Required: true}}}}

	tests := map[string]struct {
		override domain.AssistantActionDefinition
		assert   func(*testing.T, domain.AssistantActionDefinition)
	}{
		"merge-input-and-keep-hints": {
			override: domain.AssistantActionDefinition{Description: "override description", Input: domain.AssistantActionInput{Fields: map[string]domain.AssistantActionField{"max_results": {Type: "integer", Description: "max", Required: false}}}},
			assert: func(t *testing.T, got domain.AssistantActionDefinition) {
				assert.Equal(t, "override description", got.Description)
				assert.Contains(t, got.Input.Fields, "query")
				assert.Contains(t, got.Input.Fields, "max_results")
			},
		},
		"replace-approval-when-provided": {
			override: domain.AssistantActionDefinition{Approval: domain.AssistantActionApproval{Required: true, Title: "Approve action", PreviewFields: []string{"todos[].title"}, Timeout: 30 * time.Second}},
			assert: func(t *testing.T, got domain.AssistantActionDefinition) {
				assert.True(t, got.Approval.Required)
				assert.Equal(t, "Approve action", got.Approval.Title)
				assert.Equal(t, []string{"todos[].title"}, got.Approval.PreviewFields)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, mergeAssistantActionDefinition(base, tt.override))
		})
	}
}

func TestAnyToMap(t *testing.T) {
	t.Parallel()

	type sample struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	tests := map[string]struct {
		input   any
		wantOK  bool
		wantMap map[string]any
	}{
		"map-input":    {input: map[string]any{"k": "v"}, wantOK: true, wantMap: map[string]any{"k": "v"}},
		"struct-input": {input: sample{A: 1, B: "x"}, wantOK: true, wantMap: map[string]any{"a": float64(1), "b": "x"}},
		"nil-input":    {input: nil, wantOK: false},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := anyToMap(tt.input)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantMap, got)
			}
		})
	}
}

func TestAsString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input any
		want  string
	}{
		"nil":    {input: nil, want: ""},
		"string": {input: "abc", want: "abc"},
		"int":    {input: 42, want: "42"},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, asString(tt.input))
		})
	}
}
