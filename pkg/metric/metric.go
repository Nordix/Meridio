/*
Copyright (c) 2023 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metric

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Init connects to the open telemetry collector and sets the Meter provider
func Init(ctx context.Context, options ...option) (*sdkmetric.MeterProvider, error) {
	config := newConfig()
	for _, opt := range options {
		opt(config)
	}

	// connect to the ot collector since the otlp exporter is used with GRPC
	conn, err := grpc.DialContext(ctx, config.otCollectorService,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time: config.grpcKeepaliveTime,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Delta temporality is required to remove the metrics that are not being updated (e.g: a flow
	// that has been deleted should no longer be exported). As of now, otel sdk v1.16.0 supports this only
	// with the delta temporality for the counter. https://github.com/open-telemetry/opentelemetry-go/issues/3006
	// https://github.com/open-telemetry/opentelemetry-go/blob/v1.16.0/sdk/metric/internal/sum.go#L94
	deltaTemporalitySelector := func(sdkmetric.InstrumentKind) metricdata.Temporality { return metricdata.DeltaTemporality }
	metricsExporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
		otlpmetricgrpc.WithTemporalitySelector(deltaTemporalitySelector),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("meridio"),
		semconv.ServiceVersion("v1.0.0"),
	)

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricsExporter,
				sdkmetric.WithInterval(config.otCollectorInterval),
			),
		),
	)

	otel.SetMeterProvider(meterProvider)

	return meterProvider, nil
}

type config struct {
	otCollectorService  string
	grpcKeepaliveTime   time.Duration
	otCollectorInterval time.Duration
}

type option func(*config)

func newConfig() *config {
	return &config{
		otCollectorService:  "ot-collector.default:4317",
		grpcKeepaliveTime:   30 * time.Second,
		otCollectorInterval: 15 * time.Second,
	}
}

// WithOTCollectorService specifies the open telemetry Service.
// The default value is: ot-collector.default:4317
func WithOTCollectorService(otCollectorService string) option {
	return func(c *config) {
		c.otCollectorService = otCollectorService
	}
}

// WithGRPCKeepaliveTime specifies the grpc keepalive time setting for
// the grpc connection with the open telemetry collector.
// The default value is 30 seconds.
func WithGRPCKeepaliveTime(time time.Duration) option {
	return func(c *config) {
		c.grpcKeepaliveTime = time
	}
}

// WithOTCollectorInterval specifies the interval for open telemetry to
// collector the data.
// The default value is 30 seconds.
func WithOTCollectorInterval(interval time.Duration) option {
	return func(c *config) {
		c.otCollectorInterval = interval
	}
}
