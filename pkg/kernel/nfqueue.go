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

package kernel

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"github.com/nordix/meridio/pkg/networking"
	"golang.org/x/sys/unix"
)

const (
	ipv4 = 0
	ipv6 = 1

	tableName = "meridio-nfqlb"

	destinationPortSetName   = "dst-port"
	sourcePortSetName        = "src-port"
	ipv4DestinationIPSetName = "dst-ipv4"
	ipv4SourceIPSetName      = "src-ipv4"
	ipv6DestinationIPSetName = "dst-ipv6"
	ipv6SourceIPSetName      = "src-ipv6"
	protocolSetName          = "protocol"

	portRangeSeperator         = "-"
	portRangeSeperatorNFTables = "-"
)

type ipFamily int

type toSetElement func(string) []nftables.SetElement

type NFQueue struct {
	name               string
	nfqueueNumber      uint16
	priority           int32
	protocols          []string
	sourceIPv4s        []string
	destinationIPv4s   []string
	sourceIPv6s        []string
	destinationIPv6s   []string
	sourcePorts        []string
	destinationPorts   []string
	mu                 sync.Mutex
	table              *nftables.Table
	chain              *nftables.Chain
	destinationPortSet *nftables.Set
	sourcePortSet      *nftables.Set
	ipv4DestinationSet *nftables.Set
	ipv4SourceSet      *nftables.Set
	ipv6DestinationSet *nftables.Set
	ipv6SourceSet      *nftables.Set
	protocolSet        *nftables.Set
	ipv4Rule           *nftables.Rule
	ipv6Rule           *nftables.Rule
}

func NewNFQueue(name string, nfqueueNumber uint16, protocols []string, sourceIPs []string, destinationIPs []string, sourcePorts []string, destinationPorts []string, priority int32) (networking.NFQueue, error) {
	nfQueue := &NFQueue{
		name:             name,
		nfqueueNumber:    nfqueueNumber,
		priority:         priority,
		protocols:        []string{},
		sourceIPv4s:      []string{},
		destinationIPv4s: []string{},
		sourceIPv6s:      []string{},
		destinationIPv6s: []string{},
		sourcePorts:      []string{},
		destinationPorts: []string{},
	}

	err := nfQueue.configure()
	if err != nil {
		return nil, err
	}
	err = nfQueue.setSourceIPs(sourceIPs)
	if err != nil {
		return nil, err
	}
	err = nfQueue.setDestinationIPs(destinationIPs)
	if err != nil {
		return nil, err
	}
	err = nfQueue.setSourcePorts(sourcePorts)
	if err != nil {
		return nil, err
	}
	err = nfQueue.setDestinationPorts(destinationPorts)
	if err != nil {
		return nil, err
	}
	err = nfQueue.setProtocols(protocols)
	if err != nil {
		return nil, err
	}
	err = nfQueue.configureRules()
	if err != nil {
		return nil, err
	}
	return nfQueue, nil
}

func (nfq *NFQueue) Update(protocols []string, sourceIPs []string, destinationIPs []string, sourcePorts []string, destinationPorts []string) error {
	nfq.mu.Lock()
	defer nfq.mu.Unlock()
	var errFinal error
	err := nfq.setSourceIPs(sourceIPs)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = nfq.setDestinationIPs(destinationIPs)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = nfq.setSourcePorts(sourcePorts)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = nfq.setDestinationPorts(destinationPorts)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = nfq.setProtocols(protocols)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func (nfq *NFQueue) Delete() error {
	nfq.mu.Lock()
	defer nfq.mu.Unlock()
	conn := &nftables.Conn{}
	conn.DelChain(nfq.chain)
	conn.DelSet(nfq.destinationPortSet)
	conn.DelSet(nfq.sourcePortSet)
	conn.DelSet(nfq.ipv4DestinationSet)
	conn.DelSet(nfq.ipv4SourceSet)
	conn.DelSet(nfq.ipv6DestinationSet)
	conn.DelSet(nfq.ipv6SourceSet)
	conn.DelSet(nfq.protocolSet)
	return conn.Flush()
}

func (nfq *NFQueue) configure() error {
	conn := &nftables.Conn{}
	table := &nftables.Table{
		Name:   tableName,
		Family: nftables.TableFamilyINet,
	}
	nfq.table = conn.AddTable(table)

	var err error
	nfq.destinationPortSet = &nftables.Set{
		Table:    nfq.table,
		Name:     nfq.getFullSetName(destinationPortSetName),
		Interval: true,
		KeyType:  nftables.TypeInetService,
	}
	err = conn.AddSet(nfq.destinationPortSet, []nftables.SetElement{})
	if err != nil {
		return err
	}
	nfq.sourcePortSet = &nftables.Set{
		Table:    nfq.table,
		Name:     nfq.getFullSetName(sourcePortSetName),
		Interval: true,
		KeyType:  nftables.TypeInetService,
	}
	err = conn.AddSet(nfq.sourcePortSet, []nftables.SetElement{})
	if err != nil {
		return err
	}
	nfq.ipv4DestinationSet = &nftables.Set{
		Table:    nfq.table,
		Name:     nfq.getFullSetName(ipv4DestinationIPSetName),
		Interval: true,
		KeyType:  nftables.TypeIPAddr,
	}
	err = conn.AddSet(nfq.ipv4DestinationSet, []nftables.SetElement{})
	if err != nil {
		return err
	}
	nfq.ipv4SourceSet = &nftables.Set{
		Table:    nfq.table,
		Name:     nfq.getFullSetName(ipv4SourceIPSetName),
		Interval: true,
		KeyType:  nftables.TypeIPAddr,
	}
	err = conn.AddSet(nfq.ipv4SourceSet, []nftables.SetElement{})
	if err != nil {
		return err
	}
	nfq.ipv6DestinationSet = &nftables.Set{
		Table:    nfq.table,
		Name:     nfq.getFullSetName(ipv6DestinationIPSetName),
		Interval: true,
		KeyType:  nftables.TypeIP6Addr,
	}
	err = conn.AddSet(nfq.ipv6DestinationSet, []nftables.SetElement{})
	if err != nil {
		return err
	}
	nfq.ipv6SourceSet = &nftables.Set{
		Table:    nfq.table,
		Name:     nfq.getFullSetName(ipv6SourceIPSetName),
		Interval: true,
		KeyType:  nftables.TypeIP6Addr,
	}
	err = conn.AddSet(nfq.ipv6SourceSet, []nftables.SetElement{})
	if err != nil {
		return err
	}
	nfq.protocolSet = &nftables.Set{
		Table:   nfq.table,
		Name:    nfq.getFullSetName(protocolSetName),
		KeyType: nftables.TypeInetProto,
	}
	err = conn.AddSet(nfq.protocolSet, []nftables.SetElement{})
	if err != nil {
		return err
	}
	chain := &nftables.Chain{
		Name:     nfq.name,
		Table:    nfq.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriority(nfq.priority),
	}
	nfq.chain = conn.AddChain(chain)
	return conn.Flush()
}

func (nfq *NFQueue) configureRules() error {
	conn := &nftables.Conn{}
	// nft add rule inet meridio-nfqlb flow-a meta l4proto @flow-a-protocols ip saddr @flow-a-saddrs-v4 ip daddr @flow-a-daddrs-v4 th dport @flow-a-dports th sport @flow-a-sports counter queue num 1
	ipv4Rule := &nftables.Rule{
		Table: nfq.table,
		Chain: nfq.chain,
		Exprs: []expr.Any{
			// [ meta load l4proto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			// [ lookup reg 1 set flow-a-protocols ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        nfq.protocolSet.Name,
				SetID:          nfq.protocolSet.ID,
			},
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
			// [ payload load 4b @ network header + 12 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			// [ lookup reg 1 set flow-a-saddrs-v4 ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        nfq.ipv4SourceSet.Name,
				SetID:          nfq.ipv4SourceSet.ID,
			},
			// [ payload load 4b @ network header + 16 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       16,
				Len:          4,
			},
			// [ lookup reg 1 set flow-a-daddrs-v4 ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        nfq.ipv4DestinationSet.Name,
				SetID:          nfq.ipv4DestinationSet.ID,
			},
			// [ payload load 2b @ transport header + 2 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			// [ lookup reg 1 set flow-a-dports ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        nfq.destinationPortSet.Name,
				SetID:          nfq.destinationPortSet.ID,
			},
			// [ payload load 2b @ transport header + 0 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       0,
				Len:          2,
			},
			// [ lookup reg 1 set flow-a-sports ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        nfq.sourcePortSet.Name,
				SetID:          nfq.sourcePortSet.ID,
			},
			// [ counter pkts 0 bytes 0 ]
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
			// [ queue num 1 ]
			&expr.Queue{
				Num: nfq.nfqueueNumber,
			},
		},
	}
	nfq.ipv4Rule = conn.AddRule(ipv4Rule)
	// nft add rule inet meridio-nfqlb flow-a meta l4proto @flow-a-protocols ip6 saddr @flow-a-saddrs-v6 ip6 daddr @flow-a-daddrs-v6 th dport @flow-a-dports th sport @flow-a-sports counter queue num 1
	ipv6Rule := &nftables.Rule{
		Table: nfq.table,
		Chain: nfq.chain,
		Exprs: []expr.Any{
			// [ meta load l4proto => reg 1 ]
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			// [ lookup reg 1 set flow-a-protocols ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        nfq.protocolSet.Name,
				SetID:          nfq.protocolSet.ID,
			},
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
			// [ payload load 16b @ network header + 8 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       8,
				Len:          16,
			},
			// [ lookup reg 1 set flow-a-saddrs-v6 ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        nfq.ipv6SourceSet.Name,
				SetID:          nfq.ipv6SourceSet.ID,
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
				SetName:        nfq.ipv6DestinationSet.Name,
				SetID:          nfq.ipv6DestinationSet.ID,
			},
			// [ payload load 2b @ transport header + 2 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			// [ lookup reg 1 set flow-a-dports ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        nfq.destinationPortSet.Name,
				SetID:          nfq.destinationPortSet.ID,
			},
			// [ payload load 2b @ transport header + 0 => reg 1 ]
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       0,
				Len:          2,
			},
			// [ lookup reg 1 set flow-a-sports ]
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        nfq.sourcePortSet.Name,
				SetID:          nfq.sourcePortSet.ID,
			},
			// [ counter pkts 0 bytes 0 ]
			&expr.Counter{
				Bytes:   0,
				Packets: 0,
			},
			// [ queue num 1 ]
			&expr.Queue{
				Num: uint16(nfq.nfqueueNumber),
			},
		},
	}
	nfq.ipv6Rule = conn.AddRule(ipv6Rule)
	return conn.Flush()
}

func (nfq *NFQueue) setProtocols(protocols []string) error {
	p := []string{}
	for _, protocol := range protocols {
		if !validProtocol(protocol) {
			continue
		}
		p = append(p, protocol)
	}
	err := nfq.setElements(p, nfq.protocols, nfq.protocolSet, protocolToSetElement)
	nfq.protocols = p
	return err
}

func (nfq *NFQueue) setSourceIPs(sourceIPs []string) error {
	var errFinal error
	var err error
	nfq.sourceIPv4s, err = nfq.setIPs(ipv4, sourceIPs, nfq.sourceIPv4s, nfq.ipv4SourceSet)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	nfq.sourceIPv6s, err = nfq.setIPs(ipv6, sourceIPs, nfq.sourceIPv6s, nfq.ipv6SourceSet)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func (nfq *NFQueue) setDestinationIPs(destinationIPs []string) error {
	var errFinal error
	var err error
	nfq.destinationIPv4s, err = nfq.setIPs(ipv4, destinationIPs, nfq.destinationIPv4s, nfq.ipv4DestinationSet)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	nfq.destinationIPv6s, err = nfq.setIPs(ipv6, destinationIPs, nfq.destinationIPv6s, nfq.ipv6DestinationSet)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func (nfq *NFQueue) setSourcePorts(sourcePorts []string) error {
	newPorts := getValidPorts(sourcePorts)
	err := nfq.setElements(newPorts, nfq.sourcePorts, nfq.sourcePortSet, portToSetElement)
	nfq.sourcePorts = newPorts
	return err
}

func (nfq *NFQueue) setDestinationPorts(destinationPorts []string) error {
	newPorts := getValidPorts(destinationPorts)
	err := nfq.setElements(newPorts, nfq.destinationPorts, nfq.destinationPortSet, portToSetElement)
	nfq.destinationPorts = newPorts
	return err
}

func (nfq *NFQueue) getFullSetName(setName string) string {
	return fmt.Sprintf("%s-%s", nfq.name, setName)
}

func (nfq *NFQueue) setIPs(family ipFamily, newIPs []string, oldIPs []string, set *nftables.Set) ([]string, error) {
	ips := getValidIPs(family, newIPs)
	return ips, nfq.setElements(ips, oldIPs, set, ipToSetElement)
}

func (nfq *NFQueue) setElements(newElements []string, oldElements []string, set *nftables.Set, tse toSetElement) error {
	conn := &nftables.Conn{}
	var errFinal error
	toAdd := stringArrayDiff(newElements, oldElements)
	toRemove := stringArrayDiff(oldElements, newElements)
	// remove has to be before add to avoid overlapping errors
	for _, element := range toRemove {
		element := tse(element)
		err := conn.SetDeleteElements(set, element)
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
		}
	}
	for _, element := range toAdd {
		element := tse(element)
		err := conn.SetAddElements(set, element)
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
		}
	}
	err := conn.Flush()
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func ipToSetElement(ip string) []nftables.SetElement {
	_, ipNet, err := net.ParseCIDR(ip)
	if err != nil {
		return []nftables.SetElement{}
	}
	return []nftables.SetElement{
		{
			Key:         ipNet.IP,
			IntervalEnd: false,
		},
		{
			Key:         nextIP(broadcastFromIpNet(ipNet)),
			IntervalEnd: true,
		},
	}
}

func portToSetElement(port string) []nftables.SetElement {
	portUInt64, err := strconv.ParseUint(port, 10, 16)
	if err == nil { // single port
		portUInt16 := uint16(portUInt64)
		return []nftables.SetElement{
			{
				Key:         binaryutil.BigEndian.PutUint16(portUInt16),
				IntervalEnd: false,
			},
			{
				Key:         binaryutil.BigEndian.PutUint16(portUInt16 + 1),
				IntervalEnd: true,
			},
		}
	}
	// port range
	portRange := strings.Split(port, portRangeSeperator)
	if len(portRange) != 2 {
		return []nftables.SetElement{}
	}
	portUInt64Start, err0 := strconv.ParseUint(portRange[0], 10, 16)
	portUInt64End, err1 := strconv.ParseUint(portRange[1], 10, 16)
	portUInt16Start := uint16(portUInt64Start)
	portUInt16End := uint16(portUInt64End)
	if err0 != nil || err1 != nil {
		return []nftables.SetElement{}
	}
	return []nftables.SetElement{
		{
			Key:         binaryutil.BigEndian.PutUint16(portUInt16Start),
			IntervalEnd: false,
		},
		{
			Key:         binaryutil.BigEndian.PutUint16(portUInt16End + 1),
			IntervalEnd: true,
		},
	}
}

func protocolToSetElement(protocol string) []nftables.SetElement {
	prot := []byte{unix.IPPROTO_TCP}
	if protocol == strings.ToLower("udp") {
		prot = []byte{unix.IPPROTO_UDP}
	}
	return []nftables.SetElement{
		{
			Key:         prot,
			IntervalEnd: false,
		},
	}
}

func validProtocol(protocol string) bool {
	p := strings.ToLower(protocol)
	return p == "tcp" || p == "udp"
}

func getValidPorts(ports []string) []string {
	res := []string{}
	for _, port := range ports {
		_, err := strconv.ParseUint(port, 10, 16)
		if err == nil { // single port
			res = append(res, port)
			continue
		}
		// port range
		portRange := strings.Split(port, portRangeSeperator)
		if len(portRange) != 2 {
			continue
		}
		_, err0 := strconv.ParseUint(portRange[0], 10, 16)
		_, err1 := strconv.ParseUint(portRange[1], 10, 16)
		if err0 != nil || err1 != nil {
			continue
		}
		res = append(res, strings.Join(portRange, portRangeSeperatorNFTables))
	}
	return res
}

func getValidIPs(family ipFamily, ips []string) []string {
	res := []string{}
	for _, ip := range ips {
		ip, ipNet, err := net.ParseCIDR(ip)
		if err != nil {
			continue
		}
		if getIPFamily(ip) != family {
			continue
		}
		res = append(res, ipNet.String())
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

func getIPFamily(ip net.IP) ipFamily {
	if ip.To4() == nil {
		return ipv6
	}
	return ipv4
}

func broadcastFromIpNet(ipNet *net.IPNet) net.IP {
	broadcast := make([]byte, len(ipNet.IP))
	copy(broadcast, ipNet.IP)
	for i := 0; i < len(ipNet.IP); i++ {
		broadcast[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}
	return broadcast
}

func nextIP(ip net.IP) net.IP {
	next := make([]byte, len(ip))
	copy(next, ip)

	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}
