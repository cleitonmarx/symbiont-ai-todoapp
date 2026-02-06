package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("")
)

// SpanNameFormatter formats span names for HTTP requests.
// It uses the HTTP method and URL path as the span name.
func SpanNameFormatter(_ string, r *http.Request) string {
	return getHttpRoute(r)
}

func getHttpRoute(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}

// Start a new span with the global tracer.
func Start(ctx context.Context, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, getCallerName(2), opts...)
}

// RecordErrorAndStatus records an error in the span and sets the status to Error.
// Returns true if an error was recorded, false otherwise.
func RecordErrorAndStatus(span trace.Span, err error) bool {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return true
	}
	span.SetStatus(codes.Ok, "OK")
	return false
}

// HttpHandler wraps an http.Handler with OpenTelemetry instrumentation.
func HttpHandler(h http.Handler, operation string) http.Handler {
	return Middleware(operation)(h)
}

// Middleware returns an HTTP middleware that instruments handlers with OpenTelemetry.
func Middleware(operation string) func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware(
		operation,
		otelhttp.WithSpanNameFormatter(SpanNameFormatter),
		otelhttp.WithMetricAttributesFn(
			WithHttpMetricAttributes,
		),
	)
}

// getCallerName retrieves the name of the function at the specified stack depth.
func getCallerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	parts := strings.Split(fn.Name(), "/")

	return strings.ReplaceAll(parts[len(parts)-1], ".", "::")
}

// newTracerProvider creates a new tracer provider with an OTLP HTTP exporter.
func newTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, sdktrace.SpanExporter, error) {
	otlpExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(otlpExporter,
			sdktrace.WithBatchTimeout(time.Second),
		),
		sdktrace.WithResource(res),
	)
	return tracerProvider, otlpExporter, nil
}
