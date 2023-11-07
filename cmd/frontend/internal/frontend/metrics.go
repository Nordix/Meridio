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

package frontend

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/cmd/frontend/internal/bird"
	meridioMetrics "github.com/nordix/meridio/pkg/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type GatewayMetrics struct {
	meter            metric.Meter
	gateways         map[string]*nspAPI.Gateway
	metricAttributes []metric.ObserveOption
	RoutingService   *bird.RoutingService
	mu               sync.Mutex
}

func NewGatewayMetrics(metricAttributes []metric.ObserveOption) *GatewayMetrics {
	meter := otel.GetMeterProvider().Meter(meridioMetrics.METER_NAME)
	gm := &GatewayMetrics{
		meter:            meter,
		metricAttributes: metricAttributes,
	}

	return gm
}

func (gm *GatewayMetrics) Set(gateways []*nspAPI.Gateway) {
	gatewaysMap := map[string]*nspAPI.Gateway{}

	for _, gateway := range gateways {
		gatewaysMap[fmt.Sprintf("NBR-%s", gateway.Name)] = gateway
	}

	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.gateways = gatewaysMap
}

// Collect collects the metrics for the gateways.
func (gm *GatewayMetrics) Collect() error {
	if gm.RoutingService == nil {
		return errors.New("routing service not set for gateway metrics")
	}

	lp, err := gm.RoutingService.LookupCli()
	if err != nil {
		return fmt.Errorf("frontend metrics, failed to RoutingService.LookupCli: %w", err)
	}

	_, err = gm.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_ATTRACTOR_GATEWAY_ROUTES_IMPORTED,
		metric.WithUnit("routes"),
		metric.WithDescription("Counts number of routes imported for a gateway."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return gm.observe(
				ctx,
				observer,
				func(birdStats *BirdStats) int64 {
					return int64(birdStats.routesImported)
				},
				lp,
			)
		}),
	)
	if err != nil {
		return fmt.Errorf("frontend metrics, failed to Int64ObservableCounter (%s): %w", meridioMetrics.MERIDIO_ATTRACTOR_GATEWAY_ROUTES_IMPORTED, err)
	}

	_, err = gm.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_ATTRACTOR_GATEWAY_ROUTES_EXPORTED,
		metric.WithUnit("routes"),
		metric.WithDescription("Counts number of routes exported for a gateway."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return gm.observe(
				ctx,
				observer,
				func(birdStats *BirdStats) int64 {
					return int64(birdStats.routesExported)
				},
				lp,
			)
		}),
	)
	if err != nil {
		return fmt.Errorf("frontend metrics, failed to Int64ObservableCounter (%s): %w", meridioMetrics.MERIDIO_ATTRACTOR_GATEWAY_ROUTES_EXPORTED, err)
	}

	_, err = gm.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_ATTRACTOR_GATEWAY_ROUTES_PREFERRED,
		metric.WithUnit("routes"),
		metric.WithDescription("Counts number of routes preferred for a gateway."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return gm.observe(
				ctx,
				observer,
				func(birdStats *BirdStats) int64 {
					return int64(birdStats.routesPreferred)
				},
				lp,
			)
		}),
	)
	if err != nil {
		return fmt.Errorf("frontend metrics, failed to Int64ObservableCounter (%s): %w", meridioMetrics.MERIDIO_ATTRACTOR_GATEWAY_ROUTES_PREFERRED, err)
	}
	return nil
}

func (gm *GatewayMetrics) observe(ctx context.Context, observer metric.Int64Observer, valueFunc func(*BirdStats) int64, lp string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	spaOutput, err := gm.RoutingService.ShowProtocolSessions(ctx, lp, "NBR-*")
	if err != nil {
		return fmt.Errorf("frontend metrics, failed to RoutingService.ShowProtocolSessions: %w", err)
	}

	birdStats := ParseShowProtocolsAll(spaOutput)

	for _, birdStat := range birdStats {
		gateway, exists := gm.gateways[birdStat.gatewayName]
		if !exists {
			continue
		}

		metricAttributes := []metric.ObserveOption{
			metric.WithAttributes(attribute.String("Protocol", getProtocolName(gateway))),
			metric.WithAttributes(attribute.String("Gateway", gateway.Name)),
			metric.WithAttributes(attribute.String("IP", gateway.Address)),
		}
		metricAttributes = append(metricAttributes, gm.metricAttributes...)

		observer.Observe(
			valueFunc(birdStat),
			metricAttributes...,
		)
	}

	return nil
}

func getProtocolName(gateway *nspAPI.Gateway) string {
	res := "unknown"
	switch strings.ToLower(gateway.Protocol) {
	case "bgp":
		res = "BGP"
	case "static":
		res = "Static"
	}
	if gateway.Bfd {
		res = fmt.Sprintf("%s+BFD", res)
	}
	return res
}

type BirdStats struct {
	gatewayName     string
	routesImported  int
	routesExported  int
	routesPreferred int
}

func (bs *BirdStats) String() string {
	return fmt.Sprintf("%s (%d, %d, %d)", bs.gatewayName, bs.routesImported, bs.routesExported, bs.routesPreferred)
}

var regex = regexp.MustCompile(`(?m)    Routes:.*`)

func ParseShowProtocolsAll(output string) []*BirdStats {
	res := []*BirdStats{}

	protocols := strings.Split(output, "\n\n")

	for _, protocol := range protocols {
		protocolName, _, found := strings.Cut(protocol, " ")

		if !found {
			continue
		}

		newStat := &BirdStats{
			gatewayName:     protocolName,
			routesImported:  0,
			routesExported:  0,
			routesPreferred: 0,
		}

		routes := regex.FindAllString(protocol, -1)

		if len(routes) <= 0 {
			res = append(res, newStat)
			continue
		}

		routesStats := strings.ReplaceAll(routes[0], "Routes:", "")
		routesStats = strings.ReplaceAll(routesStats, "imported", "")
		routesStats = strings.ReplaceAll(routesStats, "exported", "")
		routesStats = strings.ReplaceAll(routesStats, "preferred", "")
		routesStats = strings.ReplaceAll(routesStats, " ", "")
		routesStatsSlice := strings.Split(routesStats, ",")

		if len(routesStatsSlice) != 3 {
			res = append(res, newStat)
			continue
		}

		routesImported, err := strconv.Atoi(routesStatsSlice[0])
		if err != nil {
			res = append(res, newStat)
			continue
		}

		routesExported, err := strconv.Atoi(routesStatsSlice[1])
		if err != nil {
			res = append(res, newStat)
			continue
		}

		routesPreferred, err := strconv.Atoi(routesStatsSlice[2])
		if err != nil {
			res = append(res, newStat)
			continue
		}

		newStat.routesImported = routesImported
		newStat.routesExported = routesExported
		newStat.routesPreferred = routesPreferred

		res = append(res, newStat)
	}

	return res
}
