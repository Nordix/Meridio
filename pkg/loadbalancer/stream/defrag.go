/*
Copyright (c) 2021 Nordix Foundation

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

package stream

import (
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
)

const (
	tableName   = "meridio-defrag"
	defragChain = "pre-defrag"
	inChain     = "in"
	outChain    = "out"
)

type Defrag struct {
	table            *nftables.Table
	chains           map[string]*nftables.Chain
	excludedIfPrefix string
}

// NewDefrag -
// -Load kernel's defragmentation via conntrack.
// Needed by Flow rules to match L4 information - applied on packets arriving from
// outside world.
// -Do not allow defragmentation of packets from the direction of targets.
// Thus outbound IPv4 packets can leave the LB reflecting their originating source's
// PMTU information.
// -Forbid conntrack to do "book-keeping" in order to not "litter" memory.
func NewDefrag(excludedIfPrefix string) (*Defrag, error) {
	d := &Defrag{
		excludedIfPrefix: excludedIfPrefix,
		chains:           map[string]*nftables.Chain{},
	}

	err := d.configure()
	if err != nil {
		return nil, err
	}

	err = d.setupRules()
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Defrag) configure() error {
	conn := &nftables.Conn{}
	d.table = conn.AddTable(&nftables.Table{
		Family: nftables.TableFamilyINet,
		Name:   tableName,
	})

	// skip defrag for packets arriving from the targets by registering chain before prio -400 (defrag)
	d.chains[defragChain] = conn.AddChain(&nftables.Chain{
		Name:     defragChain,
		Table:    d.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriority(-500),
	})

	// load defrag via dummy conntrack statement, but do not conntrack ingress packets
	d.chains[inChain] = conn.AddChain(&nftables.Chain{
		Name:     inChain,
		Table:    d.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriority(nftables.ChainPriorityRaw),
	})

	// do not conntrack local out packets
	d.chains[outChain] = conn.AddChain(&nftables.Chain{
		Name:     outChain,
		Table:    d.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriority(nftables.ChainPriorityRaw),
	})
	return conn.Flush()
}

func (d *Defrag) setupRules() error {
	conn := &nftables.Conn{}

	chain := d.chains[defragChain]
	if rules, _ := conn.GetRules(d.table, chain); len(rules) == 0 {
		// disable defrag via notrack for packets arriving via interfaces matching excludedIfPrefix
		conn.AddRule(&nftables.Rule{
			Table: d.table,
			Chain: chain,
			Exprs: []expr.Any{
				&expr.Meta{
					Key:      expr.MetaKeyIIFNAME,
					Register: 1,
				},
				// Note: []byte("nse") i.e. byte-stream [110 115 101] translates to "nse*",
				// while []byte("nse"+"\x00") i.e. [110 115 101 0] translates to "nse"
				&expr.Cmp{
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     []byte(d.excludedIfPrefix),
				},
				/* &expr.Counter{
					Bytes:   0,
					Packets: 0,
				}, */
				&expr.Notrack{},
			},
		})
	}

	chain = d.chains[inChain]
	if rules, _ := conn.GetRules(d.table, chain); len(rules) == 0 {
		// do not conntrack packets arriving to the POD
		conn.AddRule(&nftables.Rule{
			Table: d.table,
			Chain: chain,
			Exprs: []expr.Any{
				/* &expr.Counter{
					Bytes:   0,
					Packets: 0,
				}, */
				&expr.Notrack{},
			},
		})
		// load kernel's defrag via this dummy conntrack rule
		conn.AddRule(&nftables.Rule{
			Table: d.table,
			Chain: chain,
			Exprs: []expr.Any{
				&expr.Ct{Register: 1, SourceRegister: false, Key: expr.CtKeySTATE},
				&expr.Bitwise{
					SourceRegister: 1,
					DestRegister:   1,
					Len:            4,
					Mask:           binaryutil.NativeEndian.PutUint32(expr.CtStateBitUNTRACKED),
					Xor:            binaryutil.NativeEndian.PutUint32(0),
				},
				&expr.Cmp{Op: expr.CmpOpNeq, Register: 1, Data: []byte{0, 0, 0, 0}},
				&expr.Verdict{Kind: expr.VerdictAccept},
				/* &expr.Counter{
					Bytes:   0,
					Packets: 0,
				}, */
			},
		})
	}

	chain = d.chains[outChain]
	if rules, _ := conn.GetRules(d.table, chain); len(rules) == 0 {
		// do not conntrack packets originating from the POD
		conn.AddRule(&nftables.Rule{
			Table: d.table,
			Chain: chain,
			Exprs: []expr.Any{
				/* &expr.Counter{
					Bytes:   0,
					Packets: 0,
				}, */
				&expr.Notrack{},
			},
		})
	}

	return conn.Flush()
}

func (d *Defrag) Delete() error {
	conn := &nftables.Conn{}

	for _, chain := range d.chains {
		conn.FlushChain(chain)
		conn.DelChain(chain)
	}

	return conn.Flush()
}
