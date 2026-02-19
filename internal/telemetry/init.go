package telemetry

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// InitOpenTelemetry is a component that sets up OpenTelemetry tracing.
type InitOpenTelemetry struct {
	Logger          *log.Logger `resolve:""`
	TracesEndpoint  string      `config:"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT" default:"-"`
	MetricsEndpoint string      `config:"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT" default:"-"`
	tp              *sdktrace.TracerProvider
	se              sdktrace.SpanExporter
	mp              *sdkmetric.MeterProvider
	me              sdkmetric.Exporter
}

// Initialize sets up OpenTelemetry tracing and exporting.
func (o *InitOpenTelemetry) Initialize(ctx context.Context) (context.Context, error) {
	var err error
	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up resource.
	res, err := newAppResource(ctx)
	if err != nil {
		return ctx, err
	}

	if o.TracesEndpoint != "-" {
		// Set up trace provider.
		o.tp, o.se, err = newTracerProvider(ctx, res)
		if err != nil {
			return ctx, err
		}
		otel.SetTracerProvider(o.tp)
	}

	if o.MetricsEndpoint != "-" {
		// Set up meter provider.
		o.mp, o.me, err = newMeterProvider(ctx, res)
		if err != nil {
			return ctx, err
		}
		otel.SetMeterProvider(o.mp)
	}

	return ctx, nil
}

// Close shuts down the OpenTelemetry tracer provider and span exporter.
func (o *InitOpenTelemetry) Close() {
	if o.tp == nil && o.mp == nil {
		return
	}

	cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := o.tp.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down tracer provider: %v", err)
	}
	if err := o.se.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down span exporter: %v", err)
	}
	if err := o.mp.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down meter provider: %v", err)
	}
	if err := o.me.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down meter exporter: %v", err)
	}
}

// InitHttpClient initializes an HTTP client instrumented with OpenTelemetry
// and with retry capabilities.
type InitHttpClient struct {
	Logger *log.Logger `resolve:""`
}

func (i InitHttpClient) Initialize(ctx context.Context) (context.Context, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.RetryMax = 3
	retryClient.CheckRetry = dontRetry500StatusPolicy(retryablehttp.ErrorPropagatedRetryPolicy)
	retryClient.Logger = i.Logger

	stdClient := retryClient.StandardClient()
	stdClient.Transport = otelhttp.NewTransport(
		stdClient.Transport,
		otelhttp.WithSpanNameFormatter(SpanNameFormatter),
	)

	depend.Register(stdClient)
	return ctx, nil
}

// newPropagator creates a new composite text map propagator.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newAppResource(ctx context.Context) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("todoapp"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	return res, nil
}

// dontRetry500StatusPolicy is a retry policy for the retryablehttp client that prevents
// retries on HTTP 500 Internal Server Error responses.
func dontRetry500StatusPolicy(policy retryablehttp.CheckRetry) retryablehttp.CheckRetry {
	return func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		// do not retry on context.Canceled or context.DeadlineExceeded
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		if resp.StatusCode == http.StatusInternalServerError {
			return false, err
		}
		return policy(ctx, resp, err)
	}
}
