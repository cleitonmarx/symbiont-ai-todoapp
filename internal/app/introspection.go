package app

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/cleitonmarx/symbiont/introspection/mermaid"
)

// MermaidGraphIntrospector is an implementation of the Introspector interface that generates a Mermaid graph
// representation of the application's configuration and dependencies, and registers it in the dependency container.
type MermaidGraphIntrospector struct {
}

// Introspect generates a Mermaid graph from the provided introspection report and registers it as a named dependency.
func (i MermaidGraphIntrospector) Introspect(_ context.Context, r introspection.Report) error {
	mermaidGraph := mermaid.GenerateIntrospectionGraph(r)
	depend.RegisterNamed(mermaidGraph, "introspection-graph-mermaid")
	return nil
}
