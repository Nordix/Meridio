/*
Copyright (c) 2022 Nordix Foundation

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

package nat

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

type DestinationPortNat struct {
	Table   *nftables.Table
	PortNat *nspAPI.Conduit_PortNat
	Vips    []*nspAPI.Vip
}

func NewDestinationPortNat(table *nftables.Table, portNat *nspAPI.Conduit_PortNat) (*DestinationPortNat, error) {
	dpn := &DestinationPortNat{
		Table:   table,
		PortNat: portNat,
		Vips:    []*nspAPI.Vip{},
	}
	err := dpn.init()
	if err != nil {
		return nil, err
	}
	return dpn, dpn.SetVips(portNat.GetVips())
}

func (dpn *DestinationPortNat) init() error {
	logrus.WithFields(logrus.Fields{
		"Port":       dpn.PortNat.Port,
		"TargetPort": dpn.PortNat.TargetPort,
		"Protocol":   dpn.PortNat.Protocol,
		"VIPs":       dpn.PortNat.Vips,
	}).Infof("NAT: init")
	conn := &nftables.Conn{}
	chain := conn.AddChain(dpn.getChain())
	conn.FlushChain(chain)
	ipv4Set := dpn.getIpv4Set()
	ipv6Set := dpn.getIpv6Set()
	err := conn.AddSet(ipv4Set, nil)
	if err != nil {
		return err // shouldn't happen since elements is nil
	}
	err = conn.AddSet(ipv6Set, nil)
	if err != nil {
		return err // shouldn't happen since elements is nil
	}
	l4Proto, err := parseL4Proto(dpn.PortNat.GetProtocol())
	if err != nil {
		return err
	}
	_ = conn.AddRule(dpn.buildDestRule(
		chain, unix.AF_INET, l4Proto, ipv4Set, uint(dpn.PortNat.GetPort()), uint(dpn.PortNat.GetTargetPort())))
	_ = conn.AddRule(dpn.buildSrcRule(
		chain, unix.AF_INET, l4Proto, ipv4Set, uint(dpn.PortNat.GetPort()), uint(dpn.PortNat.GetTargetPort())))
	_ = conn.AddRule(dpn.buildDestRule(
		chain, unix.AF_INET6, l4Proto, ipv6Set, uint(dpn.PortNat.GetPort()), uint(dpn.PortNat.GetTargetPort())))
	_ = conn.AddRule(dpn.buildSrcRule(
		chain, unix.AF_INET6, l4Proto, ipv6Set, uint(dpn.PortNat.GetPort()), uint(dpn.PortNat.GetTargetPort())))
	return conn.Flush()
}

func (dpn *DestinationPortNat) SetVips(vips []*nspAPI.Vip) error {
	logrus.WithFields(logrus.Fields{
		"Port":          dpn.PortNat.Port,
		"TargetPort":    dpn.PortNat.TargetPort,
		"Protocol":      dpn.PortNat.Protocol,
		"Previous VIPs": dpn.Vips,
		"New VIPs":      vips,
	}).Infof("NAT: SetVips")
	var errFinal error
	conn := &nftables.Conn{}
	ipv4Set := dpn.getIpv4Set()
	ipv6Set := dpn.getIpv6Set()
	old := vipToSlice(dpn.Vips)
	new := vipToSlice(vips)
	toAdd := stringArrayDiff(new, old)
	toRemove := stringArrayDiff(old, new)
	for _, v := range toRemove {
		_, net, err := net.ParseCIDR(v)
		if err != nil {
			continue // ignore invalid addresses
		}

		element := []nftables.SetElement{
			{
				Key:         net.IP,
				IntervalEnd: false,
			},
			{
				Key:         endAddress(net),
				IntervalEnd: true,
			},
		}
		logrus.Debugf("NAT:SetDeleteElements:%v", element)
		if net.IP.To4() != nil {
			err = conn.SetDeleteElements(ipv4Set, element)
			if err != nil {
				logrus.Errorf("NAT:SetDeleteElements ipv4:%v", err)
				errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
			}
		} else {
			err = conn.SetDeleteElements(ipv6Set, element)
			if err != nil {
				logrus.Errorf("NAT:SetDeleteElements ipv6:%v", err)
				errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
			}
		}
	}
	for _, v := range toAdd {
		_, net, err := net.ParseCIDR(v)
		if err != nil {
			continue // ignore invalid addresses
		}

		element := []nftables.SetElement{
			{
				Key:         net.IP,
				IntervalEnd: false,
			},
			{
				Key:         endAddress(net),
				IntervalEnd: true,
			},
		}
		logrus.Debugf("NAT:SetAddElements:%v", element)
		if net.IP.To4() != nil {
			err = conn.SetAddElements(ipv4Set, element)
			if err != nil {
				logrus.Errorf("NAT:SetAddElements ipv4:%v", err)
				errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
			}
		} else {
			err = conn.SetAddElements(ipv6Set, element)
			if err != nil {
				logrus.Errorf("NAT:SetAddElements ipv6:%v", err)
				errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
			}
		}
	}
	err := conn.Flush()
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
	}
	dpn.Vips = vips
	return errFinal
}

func (dpn *DestinationPortNat) Delete() error {
	conn := &nftables.Conn{}
	conn.FlushChain(dpn.getChain())
	conn.DelChain(dpn.getChain())
	conn.DelSet(dpn.getIpv4Set())
	conn.DelSet(dpn.getIpv6Set())
	return conn.Flush()
}

func (dpn *DestinationPortNat) GetName() string {
	return dpn.PortNat.GetNatName()
}

func (dpn *DestinationPortNat) getChain() *nftables.Chain {
	return &nftables.Chain{
		Name:     dpn.GetName(),
		Table:    dpn.Table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityNATSource,
	}
}

func (dpn *DestinationPortNat) getIpv4Set() *nftables.Set {
	return &nftables.Set{
		Table:    dpn.Table,
		Name:     fmt.Sprintf("%s-ipv4", dpn.GetName()),
		Interval: true,
		KeyType:  nftables.TypeIPAddr,
	}
}

func (dpn *DestinationPortNat) getIpv6Set() *nftables.Set {
	return &nftables.Set{
		Table:    dpn.Table,
		Name:     fmt.Sprintf("%s-ipv6", dpn.GetName()),
		Interval: true,
		KeyType:  nftables.TypeIP6Addr,
	}
}

// buildDestRule builds a Rule for dport NAT (ingress)
func (dpn *DestinationPortNat) buildDestRule(
	ch *nftables.Chain, l3Proto, l4Proto byte, set *nftables.Set, dport, TargetPort uint) *nftables.Rule {

	var adrLen, adrOffset uint32
	if l3Proto == unix.AF_INET {
		adrLen = 4
		adrOffset = 16
	} else {
		adrLen = 16
		adrOffset = 24
	}

	return &nftables.Rule{
		Table: dpn.Table,
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
				Data:     encodePort(TargetPort),
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
func (dpn *DestinationPortNat) buildSrcRule(
	ch *nftables.Chain, l3Proto, l4Proto byte, set *nftables.Set, sport, TargetPort uint) *nftables.Rule {

	var adrLen, adrOffset uint32
	if l3Proto == unix.AF_INET {
		adrLen = 4
		adrOffset = 12
	} else {
		adrLen = 16
		adrOffset = 8
	}

	return &nftables.Rule{
		Table: dpn.Table,
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
				Data:     encodePort(TargetPort),
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
	switch strings.ToLower(proto) {
	case "tcp":
		return unix.IPPROTO_TCP, nil
	case "udp":
		return unix.IPPROTO_UDP, nil
	case "sctp":
		return unix.IPPROTO_SCTP, nil
	}
	return 0, fmt.Errorf("nat procotol parser: unknown protocol: %s", proto)
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

func vipToSlice(vips []*nspAPI.Vip) []string {
	res := []string{}
	for _, vip := range vips {
		res = append(res, vip.GetAddress())
	}
	return res
}

func stringArrayDiff(a []string, b []string) []string {
	diff := []string{}
	bMap := make(map[string]struct{})
	for _, item := range b {
		bMap[item] = struct{}{}
	}
	for _, item := range a {
		_, exists := bMap[item]
		if !exists {
			diff = append(diff, item)
		}
	}
	return diff
}
