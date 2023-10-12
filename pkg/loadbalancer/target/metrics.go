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

package target

import (
	"context"
	"encoding/binary"
	"sync"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	meridioMetrics "github.com/nordix/meridio/pkg/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	tableName = "meridio-metrics"
	chainName = "target-hits"
)

type HitsMetrics struct {
	hostname string
	meter    metric.Meter
	targets  map[int]*nspAPI.Target
	table    *nftables.Table
	chain    *nftables.Chain
	mu       sync.Mutex
}

func NewTargetHitsMetrics(hostname string) (*HitsMetrics, error) {
	meter := otel.GetMeterProvider().Meter(meridioMetrics.METER_NAME)
	hm := &HitsMetrics{
		hostname: hostname,
		meter:    meter,
		targets:  map[int]*nspAPI.Target{},
	}

	err := hm.init()
	if err != nil {
		return nil, err
	}

	return hm, nil
}

// init creates the nftables table and chain.
func (hm *HitsMetrics) init() error {
	conn := &nftables.Conn{}

	hm.table = conn.AddTable(&nftables.Table{
		Family: nftables.TableFamilyINet,
		Name:   tableName,
	})

	hm.chain = conn.AddChain(&nftables.Chain{
		Name:     chainName,
		Table:    hm.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPostrouting,
		Priority: nftables.ChainPriorityRef(-500),
	})

	return conn.Flush()
}

// Register adds a target as nftables rule in the postrouting chain
func (hm *HitsMetrics) Register(id int, target *nspAPI.Target) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	targetMetrics, err := hm.getMetrics()
	if err != nil {
		return err
	}

	hm.targets[id] = target

	_, exists := targetMetrics[id]
	if exists {
		return nil
	}

	conn := &nftables.Conn{}

	// nft --debug all add rule inet meridio-metrics target-hits meta mark 0x13dc counter
	// [ meta load mark => reg 1 ]
	// [ cmp eq reg 1 0x000013dc ]
	// [ counter pkts 0 bytes 0 ]
	_ = conn.AddRule(&nftables.Rule{
		Table: hm.table,
		Chain: hm.chain,
		// Handle: ,
		Exprs: []expr.Any{
			&expr.Meta{
				Key:      expr.MetaKeyMARK,
				Register: 1,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     encodeID(id),
			},
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
		},
	})

	return conn.Flush()
}

// Unregister removes the nftables rule of a target from the postrouting chain
func (hm *HitsMetrics) Unregister(id int) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.targets, id)

	targetMetrics, err := hm.getRules()
	if err != nil {
		return err
	}

	rule, exists := targetMetrics[id]
	if !exists {
		return nil
	}

	conn := &nftables.Conn{}

	err = conn.DelRule(rule)
	if err != nil {
		return err
	}

	return nil
}

// Collect collects the metrics for the all the target rules.
func (hm *HitsMetrics) Collect() error {
	_, err := hm.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_CONDUIT_STREAM_TARGET_HITS_PACKETS,
		metric.WithUnit("packets"), // TODO: what unit must be set?
		metric.WithDescription("Counts number of packets that have hit a target."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			targetMetrics, err := hm.getMetrics()
			if err != nil {
				return err
			}
			for targetID, metrics := range targetMetrics {
				// Find the registred target for the collected counter
				hm.mu.Lock()
				target, exists := hm.targets[targetID]
				hm.mu.Unlock()
				if !exists {
					continue
				}

				if target.GetStream() == nil ||
					target.GetStream().GetConduit() == nil ||
					target.GetStream().GetConduit().GetTrench() == nil {
					continue
				}

				streamName := target.GetStream().GetName()
				conduitName := target.GetStream().GetConduit().GetName()
				trenchName := target.GetStream().GetConduit().GetTrench().GetName()

				observer.Observe(
					int64(metrics.Packets),
					metric.WithAttributes(attribute.String("Hostname", hm.hostname)),
					metric.WithAttributes(attribute.String("Trench", trenchName)),
					metric.WithAttributes(attribute.String("Conduit", conduitName)),
					metric.WithAttributes(attribute.String("Stream", streamName)),
					metric.WithAttributes(attribute.Int("Identifier", targetID)),
					metric.WithAttributes(attribute.StringSlice("IPs", target.GetIps())),
				)
			}
			return nil
		}),
	)
	if err != nil {
		return err
	}

	_, err = hm.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_CONDUIT_STREAM_TARGET_HITS_BYTES,
		metric.WithUnit("bytes"), // TODO: what unit must be set?
		metric.WithDescription("Counts number of bytes that have hit a target."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			targetMetrics, err := hm.getMetrics()
			if err != nil {
				return err
			}
			for targetID, metrics := range targetMetrics {
				// Find the registred target for the collected counter
				hm.mu.Lock()
				target, exists := hm.targets[targetID]
				hm.mu.Unlock()
				if !exists {
					continue
				}

				if target.GetStream() == nil ||
					target.GetStream().GetConduit() == nil ||
					target.GetStream().GetConduit().GetTrench() == nil {
					continue
				}

				streamName := target.GetStream().GetName()
				conduitName := target.GetStream().GetConduit().GetName()
				trenchName := target.GetStream().GetConduit().GetTrench().GetName()

				observer.Observe(
					int64(metrics.Bytes),
					metric.WithAttributes(attribute.String("Hostname", hm.hostname)),
					metric.WithAttributes(attribute.String("Trench", trenchName)),
					metric.WithAttributes(attribute.String("Conduit", conduitName)),
					metric.WithAttributes(attribute.String("Stream", streamName)),
					metric.WithAttributes(attribute.Int("Identifier", targetID)),
					metric.WithAttributes(attribute.StringSlice("IPs", target.GetIps())),
				)
			}
			return nil
		}),
	)
	if err != nil {
		return err
	}

	return nil
}

// getMetrics gets all the rules in the postrouting chain and export the fwmark as key and
// the metrics/counter as value
func (hm *HitsMetrics) getMetrics() (map[int]*expr.Counter, error) {
	counters := map[int]*expr.Counter{}

	rules, err := hm.getRules()
	if err != nil {
		return nil, err
	}

	for id, rule := range rules {
		counterExpr := rule.Exprs[2].(*expr.Counter)
		if counterExpr == nil {
			continue
		}

		counters[id] = counterExpr
	}

	return counters, nil
}

func (hm *HitsMetrics) getRules() (map[int]*nftables.Rule, error) {
	conn := &nftables.Conn{}

	rules, err := conn.GetRules(hm.table, hm.chain)
	if err != nil {
		return nil, err
	}

	rulesMap := map[int]*nftables.Rule{}

	for _, rule := range rules {
		cmpExpr := rule.Exprs[1].(*expr.Cmp)
		if cmpExpr == nil {
			continue
		}

		rulesMap[decodeID(cmpExpr.Data)] = rule
	}

	return rulesMap, nil
}

func encodeID(value int) []byte {
	bs := make([]byte, 4)
	binary.NativeEndian.PutUint32(bs, uint32(value))
	return bs
}

func decodeID(value []byte) int {
	return int(binary.NativeEndian.Uint32(value))
}
