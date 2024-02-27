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
	"fmt"
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
	setName   = "fwmark-verdict"
)

type HitsMetrics struct {
	meter         metric.Meter
	targets       map[int]*nspAPI.Target
	fwmarkChains  map[int]*nftables.Chain
	table         *nftables.Table
	chain         *nftables.Chain
	fwmarkVerdict *nftables.Set
	mu            sync.Mutex
}

func NewTargetHitsMetrics() (*HitsMetrics, error) {
	meter := otel.GetMeterProvider().Meter(meridioMetrics.METER_NAME)
	hm := &HitsMetrics{
		meter:        meter,
		targets:      map[int]*nspAPI.Target{},
		fwmarkChains: map[int]*nftables.Chain{},
	}

	err := hm.init()
	if err != nil {
		return nil, err
	}

	return hm, nil
}

func (hm *HitsMetrics) Delete() error {
	conn := &nftables.Conn{}

	conn.DelTable(&nftables.Table{
		Family: nftables.TableFamilyINet,
		Name:   tableName,
	})

	return conn.Flush()
}

// init creates the nftables table and chain.
func (hm *HitsMetrics) init() error {
	_ = hm.Delete()

	conn := &nftables.Conn{}

	// nft add table inet meridio-metrics
	hm.table = conn.AddTable(&nftables.Table{
		Family: nftables.TableFamilyINet,
		Name:   tableName,
	})

	// nft 'add chain inet meridio-metrics target-hits { type filter hook postrouting priority filter ; }'
	hm.chain = conn.AddChain(&nftables.Chain{
		Name:     chainName,
		Table:    hm.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPostrouting,
		Priority: nftables.ChainPriorityRef(*nftables.ChainPriorityFilter),
	})

	err := conn.Flush()
	if err != nil {
		return fmt.Errorf("target metrics, failed to flush (add table and chain): %w", err)
	}

	// nft add map inet meridio-metrics fwmark-verdict { type mark : verdict\; }
	hm.fwmarkVerdict = &nftables.Set{
		Table:    hm.table,
		Name:     setName,
		IsMap:    true,
		KeyType:  nftables.TypeMark,
		DataType: nftables.TypeVerdict,
	}
	err = conn.AddSet(hm.fwmarkVerdict, nil)
	if err != nil {
		return fmt.Errorf("target metrics, failed to AddSet: %w", err) // shouldn't happen since elements is nil
	}

	// nft --debug all add rule inet meridio-metrics target-hits mark != 0x0 mark vmap @fwmark-verdict
	// [ meta load mark => reg 1 ]
	// [ cmp neq reg 1 0x00000000 ]
	// [ meta load mark => reg 1 ]
	// [ lookup reg 1 set fwmark-verdict dreg 0 ]
	_ = conn.AddRule(&nftables.Rule{
		Table: hm.table,
		Chain: hm.chain,
		Exprs: []expr.Any{
			&expr.Meta{
				Key:      expr.MetaKeyMARK,
				Register: 1,
			},
			&expr.Cmp{
				Op:       expr.CmpOpNeq,
				Register: 1,
				Data:     []byte{0x0},
			},
			&expr.Lookup{
				SourceRegister: 1,
				SetID:          hm.fwmarkVerdict.ID,
				SetName:        hm.fwmarkVerdict.Name,
				IsDestRegSet:   true,
				DestRegister:   0,
			},
		},
	})

	return conn.Flush()
}

// Register adds a target as nftables rule in the postrouting chain
func (hm *HitsMetrics) Register(id int, target *nspAPI.Target) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	_, exists := hm.fwmarkChains[id]
	if exists {
		return nil
	}

	hm.targets[id] = target

	conn := &nftables.Conn{}

	// nft add chain inet meridio-metrics fwmark-100
	fwmarkChain := conn.AddChain(&nftables.Chain{
		Name:  fmt.Sprintf("fwmark-%d", id),
		Table: hm.table,
	})

	// nft --debug all add rule inet meridio-metrics fwmark-100 counter
	// [ counter pkts 0 bytes 0 ]
	_ = conn.AddRule(&nftables.Rule{
		Table: hm.table,
		Chain: fwmarkChain,
		Exprs: []expr.Any{
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
		},
	})

	hm.fwmarkChains[id] = fwmarkChain

	err := conn.SetAddElements(hm.fwmarkVerdict, []nftables.SetElement{
		{
			Key: encodeID(id),
			VerdictData: &expr.Verdict{
				Kind:  expr.VerdictJump,
				Chain: fwmarkChain.Name,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("target metrics, failed to SetAddElements: %w", err)
	}

	return conn.Flush()
}

// Unregister removes the nftables rule of a target from the postrouting chain
func (hm *HitsMetrics) Unregister(id int) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	fwmarkChain, exists := hm.fwmarkChains[id]
	if !exists {
		return nil
	}

	delete(hm.targets, id)
	delete(hm.fwmarkChains, id)

	conn := &nftables.Conn{}

	err := conn.SetDeleteElements(hm.fwmarkVerdict, []nftables.SetElement{
		{
			Key: encodeID(id),
			VerdictData: &expr.Verdict{
				Kind:  expr.VerdictJump,
				Chain: fwmarkChain.Name,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("target metrics, failed to SetDeleteElements: %w", err)
	}

	conn.DelChain(fwmarkChain)

	return conn.Flush()
}

// Collect collects the metrics for the all the target rules.
func (hm *HitsMetrics) Collect() error {
	_, err := hm.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_CONDUIT_STREAM_TARGET_HIT_PACKETS,
		metric.WithUnit("packets"), // TODO: what unit must be set?
		metric.WithDescription("Counts number of packets that have hit a target."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			targetMetrics, err := hm.getMetrics()
			if err != nil {
				return fmt.Errorf("target metrics, failed to getMetrics: %w", err)
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
		return fmt.Errorf("target metrics, failed to Int64ObservableCounter: %w", err)
	}

	_, err = hm.meter.Int64ObservableCounter(
		meridioMetrics.MERIDIO_CONDUIT_STREAM_TARGET_HIT_BYTES,
		metric.WithUnit("bytes"), // TODO: what unit must be set?
		metric.WithDescription("Counts number of bytes that have hit a target."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			targetMetrics, err := hm.getMetrics()
			if err != nil {
				return fmt.Errorf("target metrics, failed to getMetrics: %w", err)
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
		return fmt.Errorf("target metrics, failed to Int64ObservableCounter: %w", err)
	}

	return nil
}

// getMetrics gets all the rules in the postrouting chain and export the fwmark as key and
// the metrics/counter as value
func (hm *HitsMetrics) getMetrics() (map[int]*expr.Counter, error) {
	counters := map[int]*expr.Counter{}
	conn := &nftables.Conn{}

	for id, chain := range hm.fwmarkChains {
		rules, err := conn.GetRules(hm.table, chain)
		if err != nil || len(rules) < 1 || len(rules[0].Exprs) < 1 {
			continue
		}

		counterExpr, ok := rules[0].Exprs[0].(*expr.Counter)
		if !ok {
			continue
		}

		counters[id] = counterExpr
	}

	return counters, nil
}

func encodeID(value int) []byte {
	bs := make([]byte, 4)
	binary.NativeEndian.PutUint32(bs, uint32(value))
	return bs
}
