package chat

import (
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListAvailableSkillsImpl_Query(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setExpectations func(*assistant.MockSkillRegistry)
		expected        []assistant.SkillDefinition
		expectedError   error
	}{
		"success": {
			setExpectations: func(m *assistant.MockSkillRegistry) {
				m.EXPECT().
					ListSkills(mock.Anything).
					Return([]assistant.SkillDefinition{
						{Name: "update_todos"},
						{Name: "web_research"},
					}, nil)
			},
			expected: []assistant.SkillDefinition{
				{Name: "update_todos"},
				{Name: "web_research"},
			},
		},
		"catalog-error": {
			setExpectations: func(m *assistant.MockSkillRegistry) {
				m.EXPECT().
					ListSkills(mock.Anything).
					Return(nil, errors.New("catalog unavailable"))
			},
			expectedError: errors.New("catalog unavailable"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockSkillRegistry := assistant.NewMockSkillRegistry(t)
			tt.setExpectations(mockSkillRegistry)

			uc := NewListAvailableSkillsImpl(mockSkillRegistry)
			got, err := uc.Query(t.Context())

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
