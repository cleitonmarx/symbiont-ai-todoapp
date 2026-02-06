package telemetry

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// WithHttpMetricAttributes returns attributes for HTTP metrics based on the request.
func WithHttpMetricAttributes(r *http.Request) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.HTTPRoute(getHttpRoute(r)),
	}
}

func newMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, sdkmetric.Exporter, error) {
	exporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure())
	if err != nil {
		return nil, nil, err
	}

	// 2. Create the MeterProvider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			exporter,
			sdkmetric.WithInterval(5*time.Second),
		)),
		// This view configures histogram aggregation for all duration instruments
		// to have specific bucket boundaries.
		// This is useful for capturing latency distributions.
		sdkmetric.WithView(sdkmetric.NewView(
			sdkmetric.Instrument{Name: "*duration*"},
			sdkmetric.Stream{
				Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
					Boundaries: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
				},
			},
		)),
	)
	return meterProvider, exporter, nil
}
