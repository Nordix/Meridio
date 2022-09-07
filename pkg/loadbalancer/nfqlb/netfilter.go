/*
Copyright (c) 2021-2022 Nordix Foundation

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

package nfqlb

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/log"
	"golang.org/x/sys/unix"
)

// netfilterAdaptor configures nftables to direct IP packets whos destination
// address matches netfilter IP Sets ipv4DestinationSet, ipv6DestinationSet to
// the configured target netfilter queue(s).
//
// Supports udpate of the IP Sets ipv4DestinationSet, ipv6DestinationSet.

/* Example config:
table inet meridio-nfqlb {
	set ipv4-vips {
		type ipv4_addr
		flags interval
		elements = { 20.0.0.1, 40.0.0.0/24 }
	}

	set ipv6-vips {
		type ipv6_addr
		flags interval
		elements = { 2000::1 }
	}

	chain nfqlb {
		type filter hook prerouting priority filter; policy accept;
		ip daddr @ipv4-vips counter packets 15364 bytes 3948540 queue num 0-3
		ip6 daddr @ipv6-vips counter packets 14800 bytes 4443820 queue num 0-3
	}

	chain nfqlb-local {
		type filter hook output priority filter; policy accept;
		meta l4proto icmp ip daddr @ipv4-vips counter packets 1 bytes 576 queue num 0-3
		meta l4proto ipv6-icmp ip6 daddr @ipv6-vips counter packets 0 bytes 0 queue num 0-3
	}
}
*/

func NewNetfilterAdaptor(options ...Option) (*netfilterAdaptor, error) {
	opts := &nfoptions{
		nfqueue: NFQueues,
	}
	for _, opt := range options {
		opt(opts)
	}

	ku := &netfilterAdaptor{
		TargetNFQueue: opts.nfqueue,
		fanout:        opts.fanout,
		table:         opts.table,
		nftqueueTotal: 1,
	}

	ku.logger = log.Logger.WithValues(
		"class", "netfilterAdaptor", "instance", ku.TargetNFQueue)
	if err := ku.configure(); err != nil {
		ku.logger.Error(err, "configure")
		return nil, err
	}

	ku.logger.V(1).Info(
		"Created", "num", ku.nftqueueNum, "total", ku.nftqueueTotal, "flag", ku.nftqueueFlag)

	return ku, nil
}

type netfilterAdaptor struct {
	TargetNFQueue      string // single number or a range e.g. "0:3"
	fanout             bool   // enable netfilter queue fanout
	table              *nftables.Table
	chain              *nftables.Chain
	localchain         *nftables.Chain
	ipv4Rule           *nftables.Rule
	ipv6Rule           *nftables.Rule
	ipv4DestinationSet *kernel.NFSetIP
	ipv6DestinationSet *kernel.NFSetIP
	nftqueueFlag       expr.QueueFlag
	nftqueueNum        uint16 // start of nqueue range
	nftqueueTotal      uint16 // number of nfqueues in use
	logger             logr.Logger
}

// Delete -
// Removes nftables chains rules
func (na *netfilterAdaptor) Delete() error {
	conn := &nftables.Conn{}
	conn.FlushChain(na.chain)
	conn.FlushChain(na.localchain)
	conn.DelChain(na.chain)
	conn.DelChain(na.localchain)
	return conn.Flush()
}

func (na *netfilterAdaptor) configure() error {
	if err := na.configureNFQueue(); err != nil {
		return err
	}

	if na.table == nil {
		// create nf table
		if err := na.configureTable(); err != nil {
			return err
		}
	}

	if err := na.configureSets(); err != nil {
		return err
	}

	if err := na.configureChainAndRules(); err != nil {
		return err
	}

	if err := na.configureLocalChainAndRules(); err != nil {
		return err
	}

	return nil
}

// configureNFQueue -
// Parses targetNFQueue to be used by nftables rules
func (na *netfilterAdaptor) configureNFQueue() error {
	nfqueues := strings.Split(na.TargetNFQueue, ":")

	num, err := strconv.ParseUint(nfqueues[0], 10, 16)
	if err != nil {
		return fmt.Errorf("netlinkAdaptor: parse nfqueue: %v", err)
	}
	na.nftqueueNum = uint16(num)

	if len(nfqueues) >= 2 {
		end, err := strconv.ParseUint(nfqueues[1], 10, 16)
		if err != nil {
			return err
		}
		na.nftqueueTotal = uint16(end - num + 1)
	}

	if na.fanout {
		na.nftqueueFlag = expr.QueueFlagFanout
	}

	return nil
}

// configureTable -
// Creates netfilter table if not yet present
func (na *netfilterAdaptor) configureTable() error {
	conn := &nftables.Conn{}

	table := conn.AddTable(&nftables.Table{
		Name:   tableName,
		Family: nftables.TableFamilyINet,
	})

	err := conn.Flush()
	if err != nil {
		return fmt.Errorf("netlinkAdaptor: nftable: %v", err)
	}

	na.table = table
	return nil
}

// configureSets -
// Creates nftables Sets for both IPv4 and IPv6 destination addresses
func (na *netfilterAdaptor) configureSets() error {
	ipv4Set, err := kernel.NewNFSetIP(ipv4VIPSetName, syscall.AF_INET, na.table)
	if err != nil {
		return fmt.Errorf("netlinkAdaptor: %v set: %v", ipv4VIPSetName, err)
	}

	ipv6Set, err := kernel.NewNFSetIP(ipv6VIPSetName, syscall.AF_INET6, na.table)
	if err != nil {
		_ = ipv4Set.Delete()
		return fmt.Errorf("netlinkAdaptor: %v set: %v", ipv6VIPSetName, err)
	}

	na.ipv4DestinationSet = ipv4Set
	na.ipv6DestinationSet = ipv6Set
	return nil
}

// configureChainAndRules -
// Adds nftables rules to direct incoming packets with matching dst address to targetNFQueue
func (na *netfilterAdaptor) configureChainAndRules() error {
	conn := &nftables.Conn{}

	na.chain = conn.AddChain(&nftables.Chain{
		Name:    chainName,
		Table:   na.table,
		Type:    nftables.ChainTypeFilter,
		Hooknum: nftables.ChainHookPrerouting,
	})

	if rules, _ := conn.GetRules(na.table, na.chain); len(rules) != 0 {
		na.logger.V(1).Info("nft chain not empty", "name", chainName, "rules", rules)
		conn.FlushChain(na.chain)
	}

	// nft add rule inet meridio-nfqlb nfqlb ip daddr @ipv4Vips counter queue num 0-3 fanout
	ipv4Rule := &nftables.Rule{
		Table: na.table,
		Chain: na.chain,
		Exprs: []expr.Any{
			// [ meta load nfproto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyNFPROTO,
				Register: 1,
			},
			// [ cmp eq reg 1 0x00000002 ]
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.AF_INET},
			},
			// [ payload load 4b @ network header + 16 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       16,
				Len:          4,
			},
			// [ lookup reg 1 set vips ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        na.ipv4DestinationSet.Name,
				SetID:          na.ipv4DestinationSet.ID,
			},
			// [ counter pkts 0 bytes 0 ]
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
			// [ queue num 1 ]
			&expr.Queue{
				Num:   na.nftqueueNum,
				Total: na.nftqueueTotal,
				Flag:  na.nftqueueFlag,
			},
		},
	}
	na.ipv4Rule = conn.AddRule(ipv4Rule)

	// nft add rule inet meridio-nfqlb nfqlb ip6 daddr @ipv6Vips counter queue num 0-3 fanout
	ipv6Rule := &nftables.Rule{
		Table: na.table,
		Chain: na.chain,
		Exprs: []expr.Any{
			// [ meta load nfproto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyNFPROTO,
				Register: 1,
			},
			// [ cmp eq reg 1 0x0000000a ]
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.AF_INET6},
			},
			// [ payload load 16b @ network header + 24 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       24,
				Len:          16,
			},
			// [ lookup reg 1 set flow-a-daddrs-v6 ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        na.ipv6DestinationSet.Name,
				SetID:          na.ipv6DestinationSet.ID,
			},
			// [ counter pkts 0 bytes 0 ]
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
			// [ queue num 1 ]
			&expr.Queue{
				Num:   na.nftqueueNum,
				Total: na.nftqueueTotal,
				Flag:  na.nftqueueFlag,
			},
		},
	}
	na.ipv6Rule = conn.AddRule(ipv6Rule)

	return conn.Flush()
}

// configureLocalChainAndRules -
// Adds nftables rules to direct locally generated ICMP unreachable reply packets
// with matching dst address to targetNFQueue (e.g. in case next-hop IP had lower PMTU)
// TODO: consider adding filter to only allow the unreachable and fragmentation related packets to match
func (na *netfilterAdaptor) configureLocalChainAndRules() error {
	conn := &nftables.Conn{}

	na.localchain = conn.AddChain(&nftables.Chain{
		Name:    localChainName,
		Table:   na.table,
		Type:    nftables.ChainTypeFilter,
		Hooknum: nftables.ChainHookOutput,
	})

	if rules, _ := conn.GetRules(na.table, na.localchain); len(rules) != 0 {
		na.logger.V(1).Info("nft chain not empty", "name", chainName, "rules", rules)
		conn.FlushChain(na.localchain)
	}

	// nft add rule inet meridio-nfqlb nfqlb-local ip meta l4proto icmp daddr @ipv4Vips counter queue num 0-3 fanout
	ipv4Rule := &nftables.Rule{
		Table: na.table,
		Chain: na.localchain,
		Exprs: []expr.Any{
			// [ meta load nfproto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyNFPROTO,
				Register: 1,
			},
			// [ cmp eq reg 1 0x00000002 ]
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.AF_INET},
			},
			// [ meta load l4proto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			// [ cmp eq reg 1 0x00000001 ]
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.IPPROTO_ICMP},
			},
			// [ payload load 4b @ network header + 16 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       16,
				Len:          4,
			},
			// [ lookup reg 1 set vips ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        na.ipv4DestinationSet.Name,
				SetID:          na.ipv4DestinationSet.ID,
			},
			// // [ payload load 2b @ transport header + 2 => reg 1 ]
			// &expr.Payload{
			// 	DestRegister: 1,
			// 	Base:         expr.PayloadBaseTransportHeader,
			// 	Offset:       0,
			// 	Len:          1,
			// },
			// // [ cmp eq reg 1 0x00000003 ]
			// &expr.Cmp{
			// 	Op:       expr.CmpOpEq,
			// 	Register: 1,
			// 	Data:     []byte{0x3},
			// },
			// [ counter pkts 0 bytes 0 ]
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
			// [ queue num 1 ]
			&expr.Queue{
				Num:   na.nftqueueNum,
				Total: na.nftqueueTotal,
				Flag:  na.nftqueueFlag,
			},
		},
	}
	na.ipv4Rule = conn.AddRule(ipv4Rule)

	// nft add rule inet meridio-nfqlb nfqlb-local ip6 meta l4proto icmpv6 daddr @ipv6Vips counter queue num 0-3 fanout
	ipv6Rule := &nftables.Rule{
		Table: na.table,
		Chain: na.localchain,
		Exprs: []expr.Any{
			// [ meta load nfproto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyNFPROTO,
				Register: 1,
			},
			// [ cmp eq reg 1 0x0000000a ]
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.AF_INET6},
			},
			// [ meta load l4proto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			// [ cmp eq reg 1 0x0000003a ]
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.IPPROTO_ICMPV6},
			},
			// [ payload load 16b @ network header + 24 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       24,
				Len:          16,
			},
			// [ lookup reg 1 set flow-a-daddrs-v6 ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        na.ipv6DestinationSet.Name,
				SetID:          na.ipv6DestinationSet.ID,
			},
			// [ counter pkts 0 bytes 0 ]
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
			// [ queue num 1 ]
			&expr.Queue{
				Num:   na.nftqueueNum,
				Total: na.nftqueueTotal,
				Flag:  na.nftqueueFlag,
			},
		},
	}
	na.ipv6Rule = conn.AddRule(ipv6Rule)

	return conn.Flush()
}

// SetDestinationIPs -
// Update nftables Set based on the VIPs so that all traffic with VIP destination
// could be handled by the user space application connected to the configured queue(s)
func (na *netfilterAdaptor) SetDestinationIPs(vips []*nspAPI.Vip) error {
	na.logger.V(2).Info("SetDestinationIPs", "vips", vips)
	ips := []string{}
	for _, vip := range vips {
		ips = append(ips, vip.Address)
	}
	if err := na.ipv4DestinationSet.Update(ips); err != nil {
		return err
	}
	if err := na.ipv6DestinationSet.Update(ips); err != nil {
		return err
	}

	return nil
}
