package http

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/cleitonmarx/symbiont/depend"
)

var (
	//go:embed templates/introspect.gohtml
	templateFS embed.FS
	tmpl       = template.Must(template.ParseFS(templateFS, "templates/introspect.gohtml"))
)

// TodoApp Introspection Graph
func IntrospectHandler(w http.ResponseWriter, r *http.Request) {
	mermaidGraph, err := depend.ResolveNamed[string]("introspection-graph-mermaid")
	if err != nil {
		http.Error(w, "Failed to resolve dependency graph", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.Execute(w, struct {
		Graph string
		Title string
	}{
		Title: "TodoApp Introspection Graph",
		Graph: mermaidGraph,
	}); err != nil {
		http.Error(w, "Failed to render introspection page", http.StatusInternalServerError)
	}
}
