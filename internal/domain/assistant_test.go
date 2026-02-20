package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssistantActionDefinition_ComposeHint(t *testing.T) {
	tests := map[string]struct {
		hints    AssistantActionHints
		wantHint string
	}{
		"all-hints-present": {
			hints: AssistantActionHints{
				UseWhen:   "the user requests a summary",
				AvoidWhen: "the input is empty",
				ArgRules:  "max_tokens must be > 0",
			},
			wantHint: "Use: the user requests a summary Avoid: the input is empty Args: max_tokens must be > 0",
		},
		"only-use-when": {
			hints: AssistantActionHints{
				UseWhen: "the user requests a summary",
			},
			wantHint: "Use: the user requests a summary",
		},
		"only-avoid-when": {
			hints: AssistantActionHints{
				AvoidWhen: "the input is empty",
			},
			wantHint: "Avoid: the input is empty",
		},
		"only-arg-rules": {
			hints: AssistantActionHints{
				ArgRules: "max_tokens must be > 0",
			},
			wantHint: "Args: max_tokens must be > 0",
		},
		"use-and-avoid": {
			hints: AssistantActionHints{
				UseWhen:   "the user requests a summary",
				AvoidWhen: "the input is empty",
			},
			wantHint: "Use: the user requests a summary Avoid: the input is empty",
		},
		"use-and-arg-rules": {
			hints: AssistantActionHints{
				UseWhen:  "the user requests a summary",
				ArgRules: "max_tokens must be > 0",
			},
			wantHint: "Use: the user requests a summary Args: max_tokens must be > 0",
		},
		"avoid-and-arg-rules": {
			hints: AssistantActionHints{
				AvoidWhen: "the input is empty",
				ArgRules:  "max_tokens must be > 0",
			},
			wantHint: "Avoid: the input is empty Args: max_tokens must be > 0",
		},
		"all-empty": {
			hints:    AssistantActionHints{},
			wantHint: "Follow the tool schema and description.",
		},
		"all-whitespace": {
			hints: AssistantActionHints{
				UseWhen:   "   ",
				AvoidWhen: "  \n  ",
				ArgRules:  "\t",
			},
			wantHint: "Follow the tool schema and description.",
		},
		"whitespace-trimmed": {
			hints: AssistantActionHints{
				UseWhen:   "  the user requests a summary  ",
				AvoidWhen: "\n  the input is empty  \n",
				ArgRules:  "\t max_tokens must be > 0 \t",
			},
			wantHint: "Use: the user requests a summary Avoid: the input is empty Args: max_tokens must be > 0",
		},
		"mixed-empty-and-nonempty": {
			hints: AssistantActionHints{
				UseWhen:   "the user requests a summary",
				AvoidWhen: "",
				ArgRules:  "max_tokens must be > 0",
			},
			wantHint: "Use: the user requests a summary Args: max_tokens must be > 0",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			def := AssistantActionDefinition{
				Hints: tt.hints,
			}
			got := def.ComposeHint()
			assert.Equal(t, tt.wantHint, got)
		})
	}
}
