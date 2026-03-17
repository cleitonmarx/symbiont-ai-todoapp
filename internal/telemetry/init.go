package telemetry

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const (
	defaultHTTPClientTimeout         = 30 * time.Second
	defaultHTTPMaxIdleConns          = 200
	defaultHTTPMaxIdleConnsPerHost   = 50
	defaultHTTPMaxConnsPerHost       = 100
	defaultHTTPIdleConnTimeout       = 90 * time.Second
	defaultHTTPTLSHandshakeTimeout   = 5 * time.Second
	defaultHTTPResponseHeaderTimeout = 15 * time.Second
	defaultHTTPExpectContinueTimeout = 1 * time.Second
	streamingHTTPMaxIdleConns        = 100
	streamingHTTPMaxIdleConnsPerHost = 20
	streamingHTTPMaxConnsPerHost     = 50
)

// InitOpenTelemetry is a component that sets up OpenTelemetry tracing.
type InitOpenTelemetry struct {
	Logger          *log.Logger `resolve:""`
	TracesEndpoint  string      `config:"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT" default:""`
	MetricsEndpoint string      `config:"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT" default:""`
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

	if o.TracesEndpoint != "" {
		// Set up trace provider.
		o.tp, o.se, err = newTracerProvider(ctx, res)
		if err != nil {
			return ctx, err
		}
		otel.SetTracerProvider(o.tp)
		tracer = otel.Tracer(tracerName)
	}

	if o.MetricsEndpoint != "" {
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

// InitHttpClient initializes instrumented outbound HTTP clients for standard and streaming workloads.
type InitHttpClient struct {
	Logger *log.Logger `resolve:""`
}

// Initialize registers an instrumented retryable HTTP client in the dependency container.
func (i InitHttpClient) Initialize(ctx context.Context) (context.Context, error) {
	stdRetryClient := newRetryClient(
		i.Logger,
		defaultHTTPMaxIdleConns,
		defaultHTTPMaxIdleConnsPerHost,
		defaultHTTPMaxConnsPerHost,
		defaultHTTPResponseHeaderTimeout,
		defaultHTTPClientTimeout,
		3,
	)
	stdClient := stdRetryClient.StandardClient()
	stdClient.Timeout = defaultHTTPClientTimeout

	streamingRetryClient := newRetryClient(
		i.Logger,
		streamingHTTPMaxIdleConns,
		streamingHTTPMaxIdleConnsPerHost,
		streamingHTTPMaxConnsPerHost,
		0,
		0,
		0,
	)
	streamingClient := streamingRetryClient.StandardClient()

	depend.RegisterNamed(stdClient, "standard")
	depend.RegisterNamed(streamingClient, "streaming")
	return ctx, nil
}

// newRetryClient creates a retryable HTTP client with an instrumented transport.
func newRetryClient(
	logger *log.Logger,
	maxIdleConns int,
	maxIdleConnsPerHost int,
	maxConnsPerHost int,
	responseHeaderTimeout time.Duration,
	timeout time.Duration,
	retryMax int,
) *retryablehttp.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.RetryMax = retryMax
	retryClient.CheckRetry = dontRetry500StatusPolicy(retryablehttp.ErrorPropagatedRetryPolicy)
	retryClient.Logger = logger
	retryClient.HTTPClient.Transport = otelhttp.NewTransport(
		newHTTPTransport(
			maxIdleConns,
			maxIdleConnsPerHost,
			maxConnsPerHost,
			responseHeaderTimeout,
		),
		otelhttp.WithSpanNameFormatter(SpanNameFormatter),
	)
	retryClient.HTTPClient.Timeout = timeout
	return retryClient
}

// newHTTPTransport creates the shared outbound transport baseline for high-concurrency workloads.
func newHTTPTransport(
	maxIdleConns int,
	maxIdleConnsPerHost int,
	maxConnsPerHost int,
	responseHeaderTimeout time.Duration,
) *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		MaxConnsPerHost:       maxConnsPerHost,
		IdleConnTimeout:       defaultHTTPIdleConnTimeout,
		TLSHandshakeTimeout:   defaultHTTPTLSHandshakeTimeout,
		ExpectContinueTimeout: defaultHTTPExpectContinueTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
	}
}

// newPropagator creates a new composite text map propagator.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newAppResource creates a new OpenTelemetry resource with environment-configured attributes.
func newAppResource(ctx context.Context) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	if resourceHasAttribute(res, semconv.ServiceInstanceIDKey) {
		return res, nil
	}

	fallbackInstanceID := resolveServiceInstanceID(os.Hostname)
	fallbackRes := resource.NewSchemaless(
		semconv.ServiceInstanceIDKey.String(fallbackInstanceID),
	)
	// Keep environment-derived attributes as authoritative when both exist.
	mergedRes, err := resource.Merge(fallbackRes, res)
	if err != nil {
		return nil, fmt.Errorf("failed to merge fallback service instance id: %w", err)
	}
	return mergedRes, nil
}

// resourceHasAttribute checks if the given resource has an attribute with the specified key.
func resourceHasAttribute(res *resource.Resource, key attribute.Key) bool {
	if res == nil {
		return false
	}

	for _, kv := range res.Attributes() {
		if kv.Key == key {
			return true
		}
	}
	return false
}

// resolveServiceInstanceID determines the fallback service instance ID for telemetry.
// It uses hostname when available and defaults to "unknown" otherwise.
func resolveServiceInstanceID(hostnameProvider func() (string, error)) string {
	if hostnameProvider != nil {
		hostname, err := hostnameProvider()
		if err == nil {
			hostname = strings.TrimSpace(hostname)
			if hostname != "" {
				return hostname
			}
		}
	}

	return "unknown"
}

// dontRetry500StatusPolicy is a retry policy for the retryablehttp client that prevents
// retries on HTTP 500 Internal Server Error responses.
func dontRetry500StatusPolicy(policy retryablehttp.CheckRetry) retryablehttp.CheckRetry {
	return func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		// do not retry on context.Canceled or context.DeadlineExceeded
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		if resp != nil && resp.StatusCode == http.StatusInternalServerError {
			return false, err
		}
		return policy(ctx, resp, err)
	}
}
