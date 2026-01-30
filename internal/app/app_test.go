package app

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/stretchr/testify/require"
)

func TestNewTodoApp_Initializers(t *testing.T) {
	app := NewTodoApp()
	require.NotNil(t, app, "NewTodoApp should not return nil")
}

func TestReportLoggerIntrospector_Introspect(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	introspector := ReportLoggerIntrospector{Logger: logger}

	// Minimal fake report
	report := introspection.Report{}
	ctx := context.Background()

	err := introspector.Introspect(ctx, report)
	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "TODOAPP INTROSPECTION REPORT")
	require.Contains(t, output, "MERMAID GRAPH")
	require.Contains(t, output, "END OF REPORT")
}
