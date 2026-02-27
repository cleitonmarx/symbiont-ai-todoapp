package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssistantActionDefinition_RequiresApproval(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		definition AssistantActionDefinition
		want       bool
	}{
		"required": {
			definition: AssistantActionDefinition{
				Approval: AssistantActionApproval{
					Required: true,
				},
			},
			want: true,
		},
		"not-required": {
			definition: AssistantActionDefinition{
				Approval: AssistantActionApproval{
					Required: false,
				},
			},
			want: false,
		},
		"zero-value": {
			definition: AssistantActionDefinition{},
			want:       false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.definition.RequiresApproval()
			assert.Equal(t, tt.want, got)
		})
	}
}
