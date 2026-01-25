package tracing

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("")
)

// SpanNameFormatter formats span names for HTTP requests.
// It uses the HTTP method and URL path as the span name.
func SpanNameFormatter(_ string, r *http.Request) string {
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

// InitOpenTelemetry is a component that sets up OpenTelemetry tracing.
type InitOpenTelemetry struct {
	Logger *log.Logger `resolve:""`
	tp     *sdktrace.TracerProvider
	se     sdktrace.SpanExporter
}

// Initialize sets up OpenTelemetry tracing and exporting.
func (o *InitOpenTelemetry) Initialize(ctx context.Context) (context.Context, error) {
	var err error
	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	o.tp, o.se, err = newTracerProvider(ctx)
	if err != nil {
		return ctx, err
	}
	otel.SetTracerProvider(o.tp)
	return ctx, nil
}

// Close shuts down the OpenTelemetry tracer provider and span exporter.
func (o *InitOpenTelemetry) Close() {
	cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := o.tp.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down tracer provider: %v", err)
	}
	if err := o.se.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down span exporter: %v", err)
	}
}

// newPropagator creates a new composite text map propagator.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newTracerProvider creates a new tracer provider with an OTLP HTTP exporter.
func newTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, sdktrace.SpanExporter, error) {
	otlpExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("todoapp"),
		),
	)
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

// InitHttpClient initializes an HTTP client instrumented with OpenTelemetry
// and with retry capabilities.
type InitHttpClient struct {
	Logger *log.Logger `resolve:""`
}

func (i InitHttpClient) Initialize(ctx context.Context) (context.Context, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.RetryMax = 5
	retryClient.Logger = i.Logger

	stdClient := retryClient.StandardClient()
	stdClient.Transport = otelhttp.NewTransport(
		stdClient.Transport,
		otelhttp.WithSpanNameFormatter(SpanNameFormatter),
	)

	depend.Register(stdClient)
	return ctx, nil
}
