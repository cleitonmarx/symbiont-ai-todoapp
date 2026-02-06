package telemetry

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestSpanNameFormatter(t *testing.T) {
	req, _ := http.NewRequest("GET", "/foo/bar", nil)
	req.Pattern = "/foo/:bar"
	assert.Equal(t, "/foo/:bar", SpanNameFormatter("", req))

	req.Pattern = ""
	assert.Equal(t, "GET /foo/bar", SpanNameFormatter("", req))
}

func TestRecordErrorAndStatus(t *testing.T) {
	span := &mockSpan{}
	err := errors.New("fail")
	assert.True(t, RecordErrorAndStatus(span, err))
	assert.Equal(t, "fail", span.lastError)
	assert.Equal(t, "fail", span.statusMsg)
	assert.Equal(t, codes.Error, span.statusCode) // codes.Error

	span = &mockSpan{}
	assert.False(t, RecordErrorAndStatus(span, nil))
	assert.Equal(t, "OK", span.statusMsg)
	assert.Equal(t, codes.Ok, span.statusCode) // codes.Ok
}

func TestStart(t *testing.T) {
	// Create in-memory exporter
	exporter := tracetest.NewInMemoryExporter()

	// Set up TracerProvider with the exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
	)
	tracer = tp.Tracer("test-tracer")

	_, span := Start(t.Context())
	span.End()

	// Assert the name
	spans := exporter.GetSpans()
	assert.Equal(t, 1, len(spans))

	assert.Equal(t, "telemetry::TestStart", spans[0].Name)

}

// --- Mocks ---

type mockSpan struct {
	trace.Span
	lastError  string
	statusCode codes.Code
	statusMsg  string
}

func (m *mockSpan) RecordError(err error, _ ...trace.EventOption) {
	m.lastError = err.Error()
}
func (m *mockSpan) SetStatus(code codes.Code, msg string) {
	m.statusCode = code
	m.statusMsg = msg
}
