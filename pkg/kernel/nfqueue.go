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
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/nordix/meridio/pkg/networking"
)

const (
	ipv4 = 0
	ipv6 = 1

	tableName = "meridio-nfqlb"

	destinationPortSet   = "dst-port"
	sourcePortSet        = "src-port"
	ipv4DestinationIPSet = "dst-ipv4"
	ipv4SourceIPSet      = "src-ipv4"
	ipv6DestinationIPSet = "dst-ipv6"
	ipv6SourceIPSet      = "src-ipv6"

	portRangeSeperator         = "-"
	portRangeSeperatorNFTables = "-"
)

type ipFamily int

type NFQueue struct {
	name             string
	nfqueueNumber    int
	priority         int
	protocols        []string
	sourceIPv4s      []string
	destinationIPv4s []string
	sourceIPv6s      []string
	destinationIPv6s []string
	sourcePorts      []string
	destinationPorts []string
	mu               sync.Mutex
}

func NewNFQueue(name string, nfqueueNumber int, protocols []string, sourceIPs []string, destinationIPs []string, sourcePorts []string, destinationPorts []string) (networking.NFQueue, error) {
	nfQueue := &NFQueue{
		name:             name,
		nfqueueNumber:    nfqueueNumber,
		priority:         0,
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
	var errFinal error
	err := execCommand(fmt.Sprintf("nft delete chain inet %s %s", tableName, nfq.name))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = execCommand(fmt.Sprintf("nft delete set inet %s %s", tableName, nfq.getFullSetName(ipv4DestinationIPSet)))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = execCommand(fmt.Sprintf("nft delete set inet %s %s", tableName, nfq.getFullSetName(ipv4SourceIPSet)))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = execCommand(fmt.Sprintf("nft delete set inet %s %s", tableName, nfq.getFullSetName(ipv6DestinationIPSet)))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = execCommand(fmt.Sprintf("nft delete set inet %s %s", tableName, nfq.getFullSetName(ipv6SourceIPSet)))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = execCommand(fmt.Sprintf("nft delete set inet %s %s", tableName, nfq.getFullSetName(destinationPortSet)))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = execCommand(fmt.Sprintf("nft delete set inet %s %s", tableName, nfq.getFullSetName(sourcePortSet)))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func (nfq *NFQueue) configure() error {
	err := execCommand(fmt.Sprintf("nft add table inet %s", tableName))
	if err != nil {
		return err
	}
	err = execCommand(fmt.Sprintf("nft add set inet %s %s { type ipv4_addr\\; flags interval\\; }", tableName, nfq.getFullSetName(ipv4DestinationIPSet)))
	if err != nil {
		return err
	}
	err = execCommand(fmt.Sprintf("nft add set inet %s %s { type ipv4_addr\\; flags interval \\; }", tableName, nfq.getFullSetName(ipv4SourceIPSet)))
	if err != nil {
		return err
	}
	err = execCommand(fmt.Sprintf("nft add set inet %s %s { type ipv6_addr\\; flags interval \\; }", tableName, nfq.getFullSetName(ipv6DestinationIPSet)))
	if err != nil {
		return err
	}
	err = execCommand(fmt.Sprintf("nft add set inet %s %s { type ipv6_addr\\; flags interval \\; }", tableName, nfq.getFullSetName(ipv6SourceIPSet)))
	if err != nil {
		return err
	}
	err = execCommand(fmt.Sprintf("nft add set inet %s %s { type inet_service\\; flags interval \\; }", tableName, nfq.getFullSetName(destinationPortSet)))
	if err != nil {
		return err
	}
	err = execCommand(fmt.Sprintf("nft add set inet %s %s { type inet_service\\; flags interval \\; }", tableName, nfq.getFullSetName(sourcePortSet)))
	if err != nil {
		return err
	}
	err = execCommand(fmt.Sprintf("nft add chain inet %s %s { type filter hook prerouting priority %d \\; }", tableName, nfq.name, nfq.priority))
	if err != nil {
		return err
	}
	return nil
}

func (nfq *NFQueue) setProtocols(protocols []string) error {
	var errFinal error
	p := []string{}
	for _, protocol := range protocols {
		if !validProtocol(protocol) {
			continue
		}
		p = append(p, protocol)
	}
	// todo: https://wiki.nftables.org/wiki-nftables/index.php/Simple_rule_management
	// toAdd := stringArrayDiff(p, nfq.protocols)
	// toRemove := stringArrayDiff(nfq.protocols, p)
	// for _, protocol := range toAdd {
	// 	// ipv4
	// 	err := execCommand(fmt.Sprintf("nft add rule inet %s %s ip protocol %s ip saddr @%s ip daddr @%s %s dport @%s %s sport @%s counter queue num %d", tableName, nfq.name, protocol, ipv4SourceIPSet, ipv4DestinationIPSet, destinationPortSet, protocol, sourcePortSet, protocol, nfq.nfqueueNumber))
	// 	if err != nil {
	// 		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	// 	}
	// 	// ipv6
	// 	err = execCommand(fmt.Sprintf("nft add rule inet %s %s ip6 nexthdr %s ip6 saddr @%s ip6 daddr @%s %s dport @%s %s sport @%s counter queue num %d", tableName, nfq.name, protocol, ipv6SourceIPSet, ipv6DestinationIPSet, destinationPortSet, protocol, sourcePortSet, protocol, nfq.nfqueueNumber))
	// 	if err != nil {
	// 		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	// 	}
	// }
	// for _, protocol := range toRemove {
	// 	// ipv4
	// 	err := execCommand(fmt.Sprintf("nft add rule inet %s %s ip protocol %s ip saddr @%s ip daddr @%s %s dport @%s %s sport @%s counter queue num %d", tableName, nfq.name, protocol, ipv4SourceIPSet, ipv4DestinationIPSet, destinationPortSet, protocol, sourcePortSet, protocol, nfq.nfqueueNumber))
	// 	if err != nil {
	// 		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	// 	}
	// 	// ipv6
	// 	err = execCommand(fmt.Sprintf("nft add rule inet %s %s ip6 nexthdr %s ip6 saddr @%s ip6 daddr @%s %s dport @%s %s sport @%s counter queue num %d", tableName, nfq.name, protocol, ipv6SourceIPSet, ipv6DestinationIPSet, destinationPortSet, protocol, sourcePortSet, protocol, nfq.nfqueueNumber))
	// 	if err != nil {
	// 		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	// 	}
	// }
	nfq.protocols = p

	err := execCommand(fmt.Sprintf("nft flush chain inet %s %s", tableName, nfq.name))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	for _, protocol := range nfq.protocols {
		// ipv4
		err := execCommand(fmt.Sprintf("nft add rule inet %s %s ip protocol %s ip saddr @%s ip daddr @%s %s dport @%s %s sport @%s counter queue num %d", tableName, nfq.name, protocol, nfq.getFullSetName(ipv4SourceIPSet), nfq.getFullSetName(ipv4DestinationIPSet), protocol, nfq.getFullSetName(destinationPortSet), protocol, nfq.getFullSetName(sourcePortSet), nfq.nfqueueNumber))
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
		}
		// ipv6
		err = execCommand(fmt.Sprintf("nft add rule inet %s %s ip6 nexthdr %s ip6 saddr @%s ip6 daddr @%s %s dport @%s %s sport @%s counter queue num %d", tableName, nfq.name, protocol, nfq.getFullSetName(ipv6SourceIPSet), nfq.getFullSetName(ipv6DestinationIPSet), protocol, nfq.getFullSetName(destinationPortSet), protocol, nfq.getFullSetName(sourcePortSet), nfq.nfqueueNumber))
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
		}
	}
	return errFinal
}

func (nfq *NFQueue) setSourceIPs(sourceIPs []string) error {
	var errFinal error
	var err error
	nfq.sourceIPv4s, err = setIPs(ipv4, sourceIPs, nfq.sourceIPv4s, nfq.getFullSetName(ipv4SourceIPSet))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	nfq.sourceIPv6s, err = setIPs(ipv6, sourceIPs, nfq.sourceIPv6s, nfq.getFullSetName(ipv6SourceIPSet))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func (nfq *NFQueue) setDestinationIPs(destinationIPs []string) error {
	var errFinal error
	var err error
	nfq.destinationIPv4s, err = setIPs(ipv4, destinationIPs, nfq.destinationIPv4s, nfq.getFullSetName(ipv4DestinationIPSet))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	nfq.destinationIPv6s, err = setIPs(ipv6, destinationIPs, nfq.destinationIPv6s, nfq.getFullSetName(ipv6DestinationIPSet))
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func (nfq *NFQueue) setSourcePorts(sourcePorts []string) error {
	newPorts := getValidPorts(sourcePorts)
	err := setElements(newPorts, nfq.sourcePorts, nfq.getFullSetName(sourcePortSet))
	nfq.sourcePorts = newPorts
	return err
}

func (nfq *NFQueue) setDestinationPorts(destinationPorts []string) error {
	newPorts := getValidPorts(destinationPorts)
	err := setElements(newPorts, nfq.destinationPorts, nfq.getFullSetName(destinationPortSet))
	nfq.destinationPorts = newPorts
	return err
}

func (nfq *NFQueue) getFullSetName(setName string) string {
	return fmt.Sprintf("%s-%s", nfq.name, setName)
}

func execCommand(cmd string) error {
	command := exec.Command("/bin/sh", "-c", cmd)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		return fmt.Errorf("%w; %s", err, stderr.String())
	}
	return nil
}

func setIPs(family ipFamily, newIPs []string, oldIPs []string, setName string) ([]string, error) {
	ips := getValidIPs(family, newIPs)
	return ips, setElements(ips, oldIPs, setName)
}

func setElements(newElements []string, oldElements []string, setName string) error {
	var errFinal error
	toAdd := stringArrayDiff(newElements, oldElements)
	toRemove := stringArrayDiff(oldElements, newElements)
	for _, ip := range toAdd {
		err := execCommand(fmt.Sprintf("nft add element inet %s %s { %s }", tableName, setName, ip))
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
		}
	}
	for _, ip := range toRemove {
		err := execCommand(fmt.Sprintf("nft delete element inet %s %s { %s }", tableName, setName, ip))
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
		}
	}
	return errFinal
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
		_, err0 := strconv.Atoi(portRange[0])
		_, err1 := strconv.Atoi(portRange[1])
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
