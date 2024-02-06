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

package flow

import (
	"context"
	"fmt"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/nfqlb"
	meridioMetrics "github.com/nordix/meridio/pkg/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// CollectMetrics collects the metrics for the flows. This function will continue
// to run until the context is cancel.
func CollectMetrics(options ...option) error {
	config := newConfig()
	for _, opt := range options {
		opt(config)
	}

	_, err := config.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_CONDUIT_STREAM_FLOW_MATCHES,
		metric.WithUnit("packets"), // TODO: what unit must be set?
		metric.WithDescription("Counts number of packets that have matched a flow."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			flowStats, err := config.getFlowStatsFunc()
			if err != nil {
				return err
			}
			for _, fs := range flowStats {
				observer.Observe(
					int64(fs.GetMatchesCount()),
					metric.WithAttributes(attribute.String("Hostname", config.hostname)),
					metric.WithAttributes(attribute.String("Trench", config.trenchName)),
					metric.WithAttributes(attribute.String("Conduit", config.conduitName)),
					metric.WithAttributes(attribute.String("Stream", fs.GetFlow().GetStream().GetName())),
					metric.WithAttributes(attribute.String("Flow", fs.GetFlow().GetName())),
				)
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create int64 observable counter (%s): %w",
			meridioMetrics.MERIDIO_CONDUIT_STREAM_FLOW_MATCHES, err)
	}

	return nil
}

type GetFlowStats func() ([]FlowStat, error)

type FlowStat interface {
	GetFlow() *nspAPI.Flow
	GetMatchesCount() int
}

func nfqlbGetFlowStats() ([]FlowStat, error) {
	list := []FlowStat{}
	nfqlbFlowStats, err := nfqlb.GetFlowStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get nfqlb flow stats: %w", err)
	}
	for _, nfqlbFlowStat := range nfqlbFlowStats {
		list = append(list, nfqlbFlowStat)
	}
	return list, nil
}

type config struct {
	getFlowStatsFunc GetFlowStats
	meter            metric.Meter
	hostname         string
	trenchName       string
	conduitName      string
}

type option func(*config)

func newConfig() *config {
	meter := otel.GetMeterProvider().Meter(meridioMetrics.METER_NAME)
	return &config{
		meter:            meter,
		getFlowStatsFunc: nfqlbGetFlowStats,
	}
}

// WithMeter specifies the meter for the metric collection.
func WithMeter(meter metric.Meter) option {
	return func(c *config) {
		c.meter = meter
	}
}

// WithGetFlowStatsFunc specifies which function will be used to get the flow metrics.
func WithGetFlowStatsFunc(getFlowStatsFunc GetFlowStats) option {
	return func(c *config) {
		c.getFlowStatsFunc = getFlowStatsFunc
	}
}

// WithHostname specifies the hostname attribute.
func WithHostname(hostname string) option {
	return func(c *config) {
		c.hostname = hostname
	}
}

// WithTrenchName specifies the trench attribute.
func WithTrenchName(trenchName string) option {
	return func(c *config) {
		c.trenchName = trenchName
	}
}

// WithConduitName specifies the conduit attribute.
func WithConduitName(conduitName string) option {
	return func(c *config) {
		c.conduitName = conduitName
	}
}
