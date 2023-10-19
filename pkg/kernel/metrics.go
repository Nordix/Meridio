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

package kernel

import (
	"context"
	"sync"

	meridioMetrics "github.com/nordix/meridio/pkg/metrics"
	"github.com/vishvananda/netlink"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type InterfaceMetrics struct {
	meter            metric.Meter
	interfaces       map[string]struct{}
	metricAttributes []metric.ObserveOption
	mu               sync.Mutex
}

func NewInterfaceMetrics(metricAttributes []metric.ObserveOption) *InterfaceMetrics {
	meter := otel.GetMeterProvider().Meter(meridioMetrics.METER_NAME)
	im := &InterfaceMetrics{
		meter:            meter,
		interfaces:       map[string]struct{}{},
		metricAttributes: metricAttributes,
	}

	return im
}

func (im *InterfaceMetrics) Register(interfaceName string) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.interfaces[interfaceName] = struct{}{}
}

func (im *InterfaceMetrics) Unregister(interfaceName string) {
	im.mu.Lock()
	defer im.mu.Unlock()
	delete(im.interfaces, interfaceName)
}

// Collect collects the metrics for the interfaces.
func (im *InterfaceMetrics) Collect() error {
	_, err := im.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_INTERFACE_RX_BYTES,
		metric.WithUnit("bytes"),
		metric.WithDescription("Counts number of received bytes for a network interface."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return im.observe(
				ctx,
				observer,
				func(metrics *netlink.LinkStatistics) int64 {
					return int64(metrics.RxBytes)
				},
			)
		}),
	)
	if err != nil {
		return err
	}

	_, err = im.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_INTERFACE_TX_BYTES,
		metric.WithUnit("bytes"),
		metric.WithDescription("Counts number of transfered bytes for a network interface."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return im.observe(
				ctx,
				observer,
				func(metrics *netlink.LinkStatistics) int64 {
					return int64(metrics.TxBytes)
				},
			)
		}),
	)
	if err != nil {
		return err
	}

	_, err = im.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_INTERFACE_RX_PACKETS,
		metric.WithUnit("packets"),
		metric.WithDescription("Counts number of received packets for a network interface."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return im.observe(
				ctx,
				observer,
				func(metrics *netlink.LinkStatistics) int64 {
					return int64(metrics.RxPackets)
				},
			)
		}),
	)
	if err != nil {
		return err
	}

	_, err = im.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_INTERFACE_TX_PACKET,
		metric.WithUnit("packets"),
		metric.WithDescription("Counts number of transfered packets for a network interface."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return im.observe(
				ctx,
				observer,
				func(metrics *netlink.LinkStatistics) int64 {
					return int64(metrics.TxPackets)
				},
			)
		}),
	)
	if err != nil {
		return err
	}

	_, err = im.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_INTERFACE_RX_ERRORS,
		metric.WithUnit("errors"),
		metric.WithDescription("Counts number of received errors for a network interface."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return im.observe(
				ctx,
				observer,
				func(metrics *netlink.LinkStatistics) int64 {
					return int64(metrics.RxErrors)
				},
			)
		}),
	)
	if err != nil {
		return err
	}

	_, err = im.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_INTERFACE_TX_ERRORS,
		metric.WithUnit("errors"),
		metric.WithDescription("Counts number of transfered errors for a network interface."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return im.observe(
				ctx,
				observer,
				func(metrics *netlink.LinkStatistics) int64 {
					return int64(metrics.TxErrors)
				},
			)
		}),
	)
	if err != nil {
		return err
	}

	_, err = im.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_INTERFACE_RX_DROPPED,
		metric.WithUnit("dropped"),
		metric.WithDescription("Counts number of received dropped for a network interface."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return im.observe(
				ctx,
				observer,
				func(metrics *netlink.LinkStatistics) int64 {
					return int64(metrics.RxDropped)
				},
			)
		}),
	)
	if err != nil {
		return err
	}

	_, err = im.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_INTERFACE_TX_DROPPED,
		metric.WithUnit("dropped"),
		metric.WithDescription("Counts number of transfered dropped for a network interface."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			return im.observe(
				ctx,
				observer,
				func(metrics *netlink.LinkStatistics) int64 {
					return int64(metrics.TxDropped)
				},
			)
		}),
	)
	if err != nil {
		return err
	}

	return nil
}

func (im *InterfaceMetrics) observe(ctx context.Context, observer metric.Int64Observer, valueFunc func(*netlink.LinkStatistics) int64) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	for interfaceName := range im.interfaces {
		metricAttributes := []metric.ObserveOption{
			metric.WithAttributes(attribute.String("Interface Name", interfaceName)),
		}
		metricAttributes = append(metricAttributes, im.metricAttributes...)
		metrics := getMetrics(interfaceName)
		if metrics == nil {
			continue
		}
		observer.Observe(
			valueFunc(metrics),
			metricAttributes...,
		)
	}
	return nil
}

func getMetrics(interfaceName string) *netlink.LinkStatistics {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil || link == nil || link.Attrs() == nil {
		return nil
	}
	return link.Attrs().Statistics
}
