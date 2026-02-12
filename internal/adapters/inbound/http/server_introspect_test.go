package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestIntrospectHandler(t *testing.T) {
	tests := map[string]struct {
		registerDependencies func(t *testing.T)
		expectedCode         int
		expectedType         string
		shouldContain        []string
	}{
		"success-returns-html": {
			registerDependencies: func(t *testing.T) {
				depend.RegisterNamed("graph TD;\nA-->B;\nB-->C;\nA-->C;", "introspection-graph-mermaid")
				t.Cleanup(depend.ClearContainer)
			},
			expectedCode: http.StatusOK,
			expectedType: "text/html; charset=utf-8",
			shouldContain: []string{
				"<!DOCTYPE html>",
				"<title>TodoApp Introspection Graph</title>",
				"mermaid.registerLayoutLoaders(elkLayouts);",
				"mermaid.initialize({ startOnLoad: false });",
				"window.addEventListener('DOMContentLoaded', renderGraph);",
				"<h1>TodoApp Introspection Graph</h1>",
				`const { svg } = await mermaid.render('mermaid-svg-id', "graph TD;\nA--\u003eB;\nB--\u003eC;\nA--\u003eC;");`,
			},
		},
		"failed-to-resolve-dependency": {
			registerDependencies: func(t *testing.T) {},
			expectedCode:         http.StatusInternalServerError,
			expectedType:         "text/plain; charset=utf-8",
			shouldContain:        []string{"Failed to resolve dependency graph"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.registerDependencies(t)

			req := httptest.NewRequest(http.MethodGet, "/introspect", nil)
			w := httptest.NewRecorder()

			IntrospectHandler(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Equal(t, tt.expectedType, w.Header().Get("Content-Type"))

			body := w.Body.String()
			for _, expectedText := range tt.shouldContain {
				assert.True(t,
					strings.Contains(body, expectedText),
					"expected response to contain %q", expectedText,
				)
			}
		})
	}
}
