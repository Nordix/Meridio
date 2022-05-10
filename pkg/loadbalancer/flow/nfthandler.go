/*
  SPDX-License-Identifier: Apache-2.0
  Copyright (c) 2022 Nordix Foundation
*/

package flow

import (
	"encoding/binary"
	"fmt"
	"net"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

/* Example config:
table inet meridio-nfqlb {

        # (things omitted...)

        set vips4-flow-port-nat {
                type ipv4_addr
                flags interval
                elements = { 10.0.0.1 }
        }

        set vips6-flow-port-nat {
                type ipv6_addr
                flags interval
                elements = { 1000::1:a00:1 }
        }

        chain flow-port-nat {
                type filter hook prerouting priority 100; policy accept;
                ip daddr @vips4-flow-port-nat tcp dport 7777 tcp dport set 5001 counter packets 305 bytes 16348 notrack
                ip saddr @vips4-flow-port-nat tcp sport 5001 tcp sport set 7777 counter packets 180 bytes 11472 notrack
                ip6 daddr @vips6-flow-port-nat tcp dport 7777 tcp dport set 5001 counter packets 600 bytes 44160 notrack
                ip6 saddr @vips6-flow-port-nat tcp sport 5001 tcp sport set 7777 counter packets 209 bytes 17496 notrack
        }
}

# Manual config;
nft list ruleset
tbl=meridio-nfqlb
chain=flow-port-nat
nft add chain inet $tbl $chain {type filter hook prerouting priority 100\;}
nft flush chain inet $tbl $chain
nft --debug all add rule inet $tbl $chain ip daddr @ipv4-vips tcp dport 7777 tcp dport set 5001 counter notrack
nft --debug all add rule inet $tbl $chain ip saddr @ipv4-vips tcp sport 5001 tcp sport set 7777 counter notrack

# The output from "nft --debug ..." is used to write the buildRule functions.
*/

// Sync with "github.com/nordix/meridio/pkg/loadbalancer/nfqlb"
// or export.
const (
	tableName = "meridio-nfqlb"
)

type nfth struct {
	table *nftables.Table
}

// NewNftHandler returns an NFT handler used for port-NAT rules
func NewNftHandler() (types.NftHandler, error) {
	conn := &nftables.Conn{}

	// Get the nft table
	tables, err := conn.ListTables()
	if err != nil {
		logrus.Errorf("NewNftHandler:ListTables:%v", err)
		return nil, err
	}
	var nfth nfth
	for _, t := range tables {
		if t.Name == tableName {
			nfth.table = t
			break
		}
	}
	if nfth.table == nil {
		err = fmt.Errorf("Table not found: %s", tableName)
		logrus.Errorf("NewNftHandler:ListTables:%v", err)
		return nil, err
	}

	return &nfth, nil
}

func (nfth *nfth) PortNATSet(
	flowName string, protocols []string, dport, localPort uint) error {
	if nfth.table == nil {
		err := fmt.Errorf("Table not found: %s", tableName)
		logrus.Errorf("PortNATSet:%v", err)
		return err
	}

	conn := &nftables.Conn{}
	ch := conn.AddChain(&nftables.Chain{
		Name:     "flow-" + flowName,
		Table:    nfth.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityNATSource,
	})
	conn.FlushChain(ch)

	ipv4Vips, ipv6Vips := nfth.vipSets(flowName)
	for _, prot := range protocols {
		l4Proto, err := parseL4Proto(prot)
		if err != nil {
			return err
		}
		_ = conn.AddRule(nfth.buildDestRule(
			ch, unix.AF_INET, l4Proto, ipv4Vips, dport, localPort))
		_ = conn.AddRule(nfth.buildSrcRule(
			ch, unix.AF_INET, l4Proto, ipv4Vips, dport, localPort))
		_ = conn.AddRule(nfth.buildDestRule(
			ch, unix.AF_INET6, l4Proto, ipv6Vips, dport, localPort))
		_ = conn.AddRule(nfth.buildSrcRule(
			ch, unix.AF_INET6, l4Proto, ipv6Vips, dport, localPort))
	}

	err := conn.Flush()
	if err != nil {
		logrus.Errorf("PortNATSet:Flush: %v", err)
	}
	return err
}

func (nfth *nfth) PortNATDelete(flowName string) {
	if ch := nfth.findChain("flow-" + flowName); ch != nil {
		conn := &nftables.Conn{}
		conn.DelChain(ch)
		_ = conn.Flush()
	}
}

// PortNATCreateSets creates address Set's for the VIP addresses in the flow.
func (nfth *nfth) PortNATCreateSets(flow *nspAPI.Flow) error {
	conn := &nftables.Conn{}
	ipv4Vips, ipv6Vips := nfth.vipSets(flow.Name)
	if err := conn.AddSet(ipv4Vips, nil); err != nil {
		return err				// shouldn't happen since elements is nil
	}
	if err := conn.AddSet(ipv6Vips, nil); err != nil {
		return err				// shouldn't happen since elements is nil
	}
	err := conn.Flush()
	if err != nil {
		logrus.Errorf("PortNATCreateSets:Flush:%v", err)
	}
	return err
}

// PortNATDeleteSets delete address Set's in the flow.
func (nfth *nfth) PortNATDeleteSets(flow *nspAPI.Flow) {
	conn := &nftables.Conn{}
	ipv4Vips, ipv6Vips := nfth.vipSets(flow.Name)
	conn.DelSet(ipv4Vips)
	conn.DelSet(ipv6Vips)
	_ = conn.Flush()
}

// PortNATSetAddresses updates addresses in the Set's for the flow
func (nfth *nfth) PortNATSetAddresses(flow *nspAPI.Flow) error {
	if len(flow.Vips) < 1 {
		return nil
	}
	conn := &nftables.Conn{}
	var err error

	ipv4Vips, ipv6Vips := nfth.vipSets(flow.Name)
	conn.FlushSet(ipv4Vips)
	conn.FlushSet(ipv6Vips)

	for _, v := range flow.Vips {
		_, net, err := net.ParseCIDR(v.Address)
		if err != nil {
			continue // ignore invalid addresses
		}

		elem := []nftables.SetElement{
			{
				Key:         net.IP,
				IntervalEnd: false,
			},
			{
				Key:         endAddress(net),
				IntervalEnd: true,
			},
		}

		logrus.Debugf("PortNATSetAddresses:SetAddElements:%v", elem)
		if net.IP.To4() != nil {
			err = conn.SetAddElements(ipv4Vips, elem)
			if err != nil {
				logrus.Errorf("PortNATSetAddresses:SetAddElements ipv4:%v", err)
				return err
			}
		} else {
			err = conn.SetAddElements(ipv6Vips, elem)
			if err != nil {
				logrus.Errorf("PortNATSetAddresses:SetAddElements ipv6:%v", err)
				return err
			}
		}
	}

	err = conn.Flush()
	if err != nil {
		logrus.Errorf("PortNATSetAddresses:Flush:%v", err)
	}
	return err
}

// vipSets return the VIP-set's for ipv4 and ipv6
func (nfth *nfth) vipSets(flowName string) (ipv4Vips, ipv6Vips *nftables.Set) {
	ipv4Vips = &nftables.Set{
		Table:    nfth.table,
		Name:     "vips4-flow-" + flowName,
		Interval: true,
		KeyType:  nftables.TypeIPAddr,
	}
	ipv6Vips = &nftables.Set{
		Table:    nfth.table,
		Name:     "vips6-flow-" + flowName,
		Interval: true,
		KeyType:  nftables.TypeIP6Addr,
	}
	return
}

// endAddress computes the end address of an address range.
// It is the address *after* the last used (for god-knows-what reason)
func endAddress(ipNet *net.IPNet) net.IP {
	endAdr := make([]byte, len(ipNet.IP))
	copy(endAdr, ipNet.IP)
	for i := 0; i < len(ipNet.IP); i++ {
		endAdr[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}
	for i := len(endAdr) - 1; i >= 0; i-- {
		endAdr[i]++
		if endAdr[i] != 0 {
			break
		}
	}
	return endAdr
}

// findChain is a GetChainByName() which is lacking in the nftables package
func (nfth *nfth) findChain(chain string) *nftables.Chain {
	if nfth.table == nil {
		return nil
	}
	conn := &nftables.Conn{}
	chains, err := conn.ListChains()
	if err != nil {
		return nil
	}
	for _, ch := range chains {
		if ch.Name == chain && ch.Table.Name == nfth.table.Name {
			return ch
		}
	}
	return nil
}

// buildDestRule builds a Rule for dport NAT (ingress)
func (nfth *nfth) buildDestRule(
	ch *nftables.Chain, l3Proto, l4Proto byte, set *nftables.Set, dport, localPort uint) *nftables.Rule {

	var adrLen, adrOffset uint32
	if l3Proto == unix.AF_INET {
		adrLen = 4
		adrOffset = 16
	} else {
		adrLen = 16
		adrOffset = 24
	}

	return &nftables.Rule{
		Table: nfth.table,
		Chain: ch,
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
				Data:     []byte{l3Proto},
			},
			// [ payload load 4b @ network header + 16 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       adrOffset,
				Len:          adrLen,
			},
			// [ lookup reg 1 set vips ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        set.Name,
				SetID:          set.ID,
			},
			// [ meta load l4proto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			// [ cmp eq reg 1 0x00000006 ]
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{l4Proto},
			},
			// [ payload load 2b @ transport header + 2 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			// [ cmp eq reg 1 0x0000611e ] 1e61 = 7777
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     encodePort(dport),
			},
			// [ immediate reg 1 0x00008913 ] 1389 = 5001
			&expr.Immediate{
				Register: 1,
				Data:     encodePort(localPort),
			},
			// [ payload write reg 1 => 2b @ transport header + 2 csum_type 1 csum_off 16 csum_flags 0x0 ]
			&expr.Payload{
				OperationType:  expr.PayloadWrite,
				SourceRegister: 1,
				Base:           expr.PayloadBaseTransportHeader,
				Offset:         2,
				Len:            2,
				CsumType:       expr.CsumTypeInet,
				CsumOffset:     16,
			},
			// [ counter pkts 0 bytes 0 ]
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
			// [ notrack ]
			&expr.Notrack{},
		},
	}
}

// buildSrcRule builds a Rule for sport NAT (egress)
func (nfth *nfth) buildSrcRule(
	ch *nftables.Chain, l3Proto, l4Proto byte, set *nftables.Set, sport, localPort uint) *nftables.Rule {

	var adrLen, adrOffset uint32
	if l3Proto == unix.AF_INET {
		adrLen = 4
		adrOffset = 12
	} else {
		adrLen = 16
		adrOffset = 8
	}

	return &nftables.Rule{
		Table: nfth.table,
		Chain: ch,
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
				Data:     []byte{l3Proto},
			},
			// [ payload load 4b @ network header + 12 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       adrOffset,
				Len:          adrLen,
			},
			// [ lookup reg 1 set vips ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        set.Name,
				SetID:          set.ID,
			},
			// [ meta load l4proto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			// [ cmp eq reg 1 0x00000006 ]
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{l4Proto},
			},
			// [ payload load 2b @ transport header + 0 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       0,
				Len:          2,
			},
			// [ cmp eq reg 1 0x0000611e ] 1e61 = 7777
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     encodePort(localPort),
			},
			// [ immediate reg 1 0x00008913 ] 1389 = 5001
			&expr.Immediate{
				Register: 1,
				Data:     encodePort(sport),
			},
			// [ payload write reg 1 => 2b @ transport header + 0 csum_type 1 csum_off 16 csum_flags 0x0 ]
			&expr.Payload{
				OperationType:  expr.PayloadWrite,
				SourceRegister: 1,
				Base:           expr.PayloadBaseTransportHeader,
				Offset:         0,
				Len:            2,
				CsumType:       expr.CsumTypeInet,
				CsumOffset:     16,
			},
			// [ counter pkts 0 bytes 0 ]
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
			// [ notrack ]
			&expr.Notrack{},
		},
	}
}

// encodePort encodes a port to a 16-bit int in network byte order
func encodePort(value uint) []byte {
	bs := make([]byte, 2)
	binary.BigEndian.PutUint16(bs, uint16(value))
	return bs
}

// parseL4Proto parses supported port-NAT protocols
func parseL4Proto(proto string) (byte, error) {
	switch proto {
	case "tcp":
		return unix.IPPROTO_TCP, nil
	case "udp":
		return unix.IPPROTO_UDP, nil
	case "sctp":
		return unix.IPPROTO_SCTP, nil
	}
	return 0, fmt.Errorf("Unknown protocol: %s", proto)
}
