package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Metrics holds all application metrics
type Metrics struct {
	RequestsTotal       metric.Int64Counter
	RequestDuration     metric.Float64Histogram
	RateLimitRejections metric.Int64Counter
}

// InitMetrics initializes OpenTelemetry metrics with OTLP gRPC exporter
func InitMetrics(ctx context.Context, serviceName, endpoint string) (*sdkmetric.MeterProvider, *Metrics, error) {
	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP gRPC exporter
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
		otlpmetricgrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	// Create meter provider
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(30*time.Second),
		)),
	)

	// Set global meter provider
	otel.SetMeterProvider(mp)

	// Create meters
	meter := mp.Meter("linkko-api")

	// Create metrics
	requestsTotal, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create requests counter: %w", err)
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create duration histogram: %w", err)
	}

	rateLimitRejections, err := meter.Int64Counter(
		"rate_limit_rejections_total",
		metric.WithDescription("Total number of rate limit rejections"),
		metric.WithUnit("{rejection}"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rate limit counter: %w", err)
	}

	metrics := &Metrics{
		RequestsTotal:       requestsTotal,
		RequestDuration:     requestDuration,
		RateLimitRejections: rateLimitRejections,
	}

	return mp, metrics, nil
}
