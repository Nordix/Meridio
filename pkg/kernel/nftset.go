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

package kernel

import (
	"fmt"
	"net"
	"sync"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/google/nftables"
	"github.com/nordix/meridio/pkg/log"
)

const (
	ipv4 = syscall.AF_INET
	ipv6 = syscall.AF_INET6
)

type toSetElement func(string) []nftables.SetElement

// NFSetIP is a wrapper for IP type nftables.Set (either IPv4 or IPv6)
// Supports update of IPs
type NFSetIP struct {
	*nftables.Set
	table      *nftables.Table
	ipAdresses []string
	name       string
	family     int
	mu         sync.Mutex
	logger     logr.Logger
}

func NewNFSetIP(name string, family int, table *nftables.Table) (*NFSetIP, error) {
	nfSet := &NFSetIP{
		name:       name,
		family:     family,
		table:      table,
		ipAdresses: []string{},
		logger:     log.Logger.WithValues("class", "NFSetIP", "instance", name),
	}

	err := nfSet.configure()
	if err != nil {
		return nil, err
	}
	return nfSet, nil
}

func (nfs *NFSetIP) Update(ips []string) error {
	nfs.mu.Lock()
	defer nfs.mu.Unlock()
	var errFinal error
	nfs.logger.V(2).Info("Update", "IPs", ips)
	err := nfs.setIPs(ips)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func (nfs *NFSetIP) Delete() error {
	nfs.mu.Lock()
	defer nfs.mu.Unlock()
	nfs.logger.V(1).Info("Delete")
	conn := &nftables.Conn{}
	conn.DelSet(nfs.Set)
	err := conn.Flush()
	if err != nil {
		return fmt.Errorf("failed flusing while deleting NFSetIP (set: %s ; table: %s): %w", nfs.Set.Name, nfs.table.Name, err)
	}

	return nil
}

func (nfs *NFSetIP) configure() error {
	conn := &nftables.Conn{}
	var err error

	nfs.Set = &nftables.Set{
		Table:    nfs.table,
		Name:     nfs.name,
		Interval: true,
		KeyType: func() nftables.SetDatatype {
			switch nfs.family {
			case ipv4:
				return nftables.TypeIPAddr

			case ipv6:
				return nftables.TypeIP6Addr
			}
			return nftables.TypeInvalid
		}(),
	}
	err = conn.AddSet(nfs.Set, []nftables.SetElement{})
	if err != nil {
		return fmt.Errorf("failed AddSet while configuring NFSetIP (set: %s ; table: %s): %w", nfs.Set.Name, nfs.table.Name, err)
	}

	err = conn.Flush()
	if err != nil {
		return fmt.Errorf("failed flusing while configuring NFSetIP (set: %s ; table: %s): %w", nfs.Set.Name, nfs.table.Name, err)
	}

	return nil
}

func (nfs *NFSetIP) setIPs(ips []string) error {
	var errFinal error
	var err error
	nfs.logger.V(2).Info("setIPs", "IPs", ips)
	nfs.ipAdresses, err = nfs.updateIPs(nfs.family, ips, nfs.ipAdresses, nfs.Set)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func (nfs *NFSetIP) updateIPs(family int, newIPs []string, oldIPs []string, set *nftables.Set) ([]string, error) {
	nfs.logger.V(2).Info("updateIPs", "newIPs", newIPs, "oldIPs", oldIPs)
	ips := getValidIPs(family, newIPs)
	return ips, setElements(ips, oldIPs, set, ipToSetElement)
}

func setElements(newElements []string, oldElements []string, set *nftables.Set, tse toSetElement) error {
	logger := log.Logger.WithValues("func", "setElements")
	logger.V(2).Info("Called", "newElements", newElements, "oldElements", oldElements, "set", set)
	conn := &nftables.Conn{}
	var errFinal error
	toAdd := stringArrayDiff(newElements, oldElements)
	toRemove := stringArrayDiff(oldElements, newElements)
	// remove has to be before add to avoid overlapping errors
	for _, element := range toRemove {
		element := tse(element)
		err := conn.SetDeleteElements(set, element)
		if err != nil {
			logger.V(1).Info("SetDeleteElements", "error", err)
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
		}
	}
	for _, element := range toAdd {
		element := tse(element)
		err := conn.SetAddElements(set, element)
		if err != nil {
			logger.V(1).Info("SetAddElements", "error", err)
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
		}
	}
	err := conn.Flush()
	if err != nil {
		logger.V(1).Info("Flush", "error", err)
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

func getValidIPs(family int, ips []string) []string {
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

func getIPFamily(ip net.IP) int {
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
