//go:build skillmatrix

package skillsmatrix

import (
	"net/http"
	"os"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/modelrunner"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/skillregistry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type skillMatrixCase struct {
	messages    []assistant.Message
	summary     string
	wantTop     string
	wantContain []string
}

func runSkillMatrixCase(t *testing.T, registry assistant.SkillRegistry, tc skillMatrixCase) {
	t.Helper()

	got := registry.ListRelevant(t.Context(), assistant.SkillQueryContext{
		Messages:            tc.messages,
		ConversationSummary: tc.summary,
	})

	if tc.wantTop == "" {
		assert.Empty(t, got)
		return
	}

	require.NotEmpty(t, got)
	assert.Equal(t, tc.wantTop, got[0].Name)

	gotNames := make([]string, 0, len(got))
	for _, skill := range got {
		gotNames = append(gotNames, skill.Name)
	}
	for _, want := range tc.wantContain {
		assert.Contains(t, gotNames, want)
	}
}

func newSkillMatrixRegistry(t *testing.T) assistant.SkillRegistry {
	t.Helper()

	ctx := t.Context()

	skills, err := skillregistry.LoadSkillsFromFS(os.DirFS("../../internal/adapters/outbound/skillregistry/skills"))
	require.NoError(t, err)

	drmClient := modelrunner.NewDRMAPIClient("http://localhost:12434", "", http.DefaultClient)
	encoder := modelrunner.NewAssistantClientAdapter(drmClient, drmClient)

	registry, err := skillregistry.NewSkillRegistry(ctx, skills, encoder, "embeddinggemma:300M-Q8_0", skillregistry.Config{
		RelevantSkillsTopK:     2,
		RelevantSkillsMinScore: 0.23,
		AvoidPenaltyWeight:     0.70,
		AvoidBlockThreshold:    0.45,
		StrongUseWhenScore:     0.55,
		CurrentInputWeight:     0.70,
		RecentInputsWeight:     0.25,
		SummaryWeight:          0.05,
		RecentInputsLimit:      4,
		LogScores:              true,
	})
	require.NoError(t, err)

	return registry
}
