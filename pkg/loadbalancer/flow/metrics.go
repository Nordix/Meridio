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
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/nfqlb"
	meridioMetric "github.com/nordix/meridio/pkg/metric"
	"github.com/nordix/meridio/pkg/retry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// CollectMetrics collects the metrics for the flows. This function will continue
// to run until the context is cancel.
func CollectMetrics(ctx context.Context, options ...option) error {
	config := newConfig()
	for _, opt := range options {
		opt(config)
	}

	counter, err := registerCounter(config.meter)
	if err != nil {
		return err
	}

	// Used to save the previous data recorded got from getFlowStatsFunc.
	// It is required to do the difference with the newer value.
	flowStatsMap := map[string]int{}

	_ = retry.Do(func() error {
		flowStats, err := config.getFlowStatsFunc()
		if err != nil {
			return err
		}
		flowStatsMapNew := map[string]int{}
		for _, fs := range flowStats {
			previous, exists := flowStatsMap[flowStatName(fs)]
			if !exists {
				previous = 0
			}
			// A counter instrument requires the difference between the previous and newer metric value.
			diff := int64(fs.GetMatchesCount() - previous)
			counter.Add(
				ctx,
				diff,
				metric.WithAttributes(attribute.String("Hostname", config.hostname)),
				metric.WithAttributes(attribute.String("Trench", config.trenchName)),
				metric.WithAttributes(attribute.String("Conduit", config.conduitName)),
				metric.WithAttributes(attribute.String("Stream", fs.GetFlow().GetStream().GetName())),
				metric.WithAttributes(attribute.String("Flow", fs.GetFlow().GetName())),
			)
			flowStatsMapNew[flowStatName(fs)] = fs.GetMatchesCount()
		}
		flowStatsMap = flowStatsMapNew
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(config.interval),
		retry.WithErrorIngnored())
	return nil
}

func registerCounter(meter metric.Meter) (metric.Int64Counter, error) {
	return meter.Int64Counter(
		meridioMetric.MERIDIO_CONDUIT_STREAM_FLOW_MATCHES,
		metric.WithUnit("packets"), // TODO: what unit must be set?
		metric.WithDescription("Counts number of packets that have matched a flow."),
	)
}

type GetFlowStats func() ([]FlowStat, error)

type FlowStat interface {
	GetFlow() *nspAPI.Flow
	GetMatchesCount() int
}

func nfqlbGetFlowStats() ([]FlowStat, error) {
	list := []FlowStat{}
	nfqlbFlowStats, err := nfqlb.GetFlowStats()
	for _, nfqlbFlowStat := range nfqlbFlowStats {
		list = append(list, nfqlbFlowStat)
	}
	return list, err
}

func flowStatName(fs FlowStat) string {
	return fmt.Sprintf("%s.%s", fs.GetFlow().GetName(), fs.GetFlow().GetStream().GetName())
}

type config struct {
	getFlowStatsFunc GetFlowStats
	meter            metric.Meter
	hostname         string
	trenchName       string
	conduitName      string
	interval         time.Duration
}

type option func(*config)

func newConfig() *config {
	meter := otel.GetMeterProvider().Meter(meridioMetric.METER_NAME)
	return &config{
		meter:            meter,
		getFlowStatsFunc: nfqlbGetFlowStats,
		interval:         10 * time.Second,
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

// WithInterval specifies interval between the metrics collection.
func WithInterval(interval time.Duration) option {
	return func(c *config) {
		c.interval = interval
	}
}
