package app

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/stretchr/testify/require"
)

func TestMermaidGraphIntrospector_Introspect(t *testing.T) {
	introspector := MermaidGraphIntrospector{}

	report := introspection.Report{
		Configs: []introspection.ConfigAccess{
			{
				Key:         "KEY1",
				UsedDefault: true,
			},
		},
	}
	ctx := context.Background()

	err := introspector.Introspect(ctx, report)
	require.NoError(t, err)
	mermaidGraph, err := depend.ResolveNamed[string]("introspection-graph-mermaid")
	require.NoError(t, err)
	require.NotEmpty(t, mermaidGraph, "Mermaid graph should be registered as a named dependency")
}
