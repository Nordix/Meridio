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

	"github.com/nordix/meridio/pkg/networking"
)

const (
	ipv4 = 0
	ipv6 = 1
)

const (
	add    = "-A"
	remove = "-D"
)

const (
	portRangeSeperator         = "-"
	portRangeSeperatorIpTables = ":"
	listSeperatorIpTables      = ","
)

type ipFamily int
type action string

type NFQueue struct {
	nfqueueNumber    int
	protocols        []string
	sourceIPv4s      []string
	destinationIPv4s []string
	sourceIPv6s      []string
	destinationIPv6s []string
	sourcePorts      []string
	destinationPorts []string
}

func NewNFQueue(nfqueueNumber int, protocols []string, sourceIPs []string, destinationIPs []string, sourcePorts []string, destinationPorts []string) (networking.NFQueue, error) {
	nfQueue := &NFQueue{
		nfqueueNumber: nfqueueNumber,
	}
	nfQueue.setProtocols(protocols)
	nfQueue.setSourceIPs(sourceIPs)
	nfQueue.setDestinationIPs(destinationIPs)
	nfQueue.setSourcePorts(sourcePorts)
	nfQueue.setDestinationPorts(destinationPorts)
	err := nfQueue.configure(add)
	if err != nil {
		return nil, err
	}
	return nfQueue, nil
}

func (nfq *NFQueue) Update(protocols []string, sourceIPs []string, destinationIPs []string, sourcePorts []string, destinationPorts []string) error {
	err := nfq.Delete()
	if err != nil {
		return err
	}
	nfq.setProtocols(protocols)
	nfq.setSourceIPs(sourceIPs)
	nfq.setDestinationIPs(destinationIPs)
	nfq.setSourcePorts(sourcePorts)
	nfq.setDestinationPorts(destinationPorts)
	err = nfq.configure(add)
	if err != nil {
		return err
	}
	return nil
}

func (nfq *NFQueue) Delete() error {
	return nfq.configure(remove)
}

func (nfq *NFQueue) configure(act action) error {
	ipv4SourceIPs := formatIPs(nfq.sourceIPv4s)
	ipv6SourceIPs := formatIPs(nfq.sourceIPv6s)
	ipv4DestinationIPs := formatIPs(nfq.destinationIPv4s)
	ipv6DestinationIPs := formatIPs(nfq.destinationIPv6s)
	formattedsourcePorts := formatPorts(nfq.sourcePorts)
	if len(nfq.sourcePorts) <= 0 {
		return nil
	}
	if len(nfq.destinationPorts) <= 0 {
		return nil
	}
	var errFinal error
	for _, protocol := range nfq.protocols {
		for _, destinationPort := range nfq.destinationPorts {
			// ipv4
			if len(nfq.sourceIPv4s) > 0 && len(nfq.destinationIPv4s) > 0 {
				err := nfq.apply(act, ipv4, protocol, ipv4SourceIPs, ipv4DestinationIPs, formattedsourcePorts, destinationPort)
				if err != nil {
					errFinal = fmt.Errorf("%w; ipv4 %v", errFinal, err) // todo
				}
			}
			// ipv6
			if len(nfq.sourceIPv6s) > 0 && len(nfq.destinationIPv6s) > 0 {
				err := nfq.apply(act, ipv6, protocol, ipv6SourceIPs, ipv6DestinationIPs, formattedsourcePorts, destinationPort)
				if err != nil {
					errFinal = fmt.Errorf("%w; ipv6 %v", errFinal, err) // todo
				}
			}
		}
	}
	return errFinal
}

// iptables -t mangle -A PREROUTING -p tcp --dport 80:90 --match multiport --sports 80,90:100 -s 10.0.1.1/24,20.0.1.1/24 -d 50.0.0.1,51.0.0.1/32 -j NFQUEUE --queue-num 2
func (nfq *NFQueue) apply(act action, family ipFamily, protocol string, sourceIPs string, destinationIPs string, sourcePorts string, destinationPort string) error {
	ipTablesCmd := nfq.getCommand(act, family, protocol, sourceIPs, destinationIPs, sourcePorts, destinationPort)
	var stderr bytes.Buffer
	ipTablesCmd.Stderr = &stderr
	_, err := ipTablesCmd.Output()
	if err != nil {
		return fmt.Errorf("%w; %s", err, stderr.String())
	}
	return nil
}

func (nfq *NFQueue) getCommand(act action, family ipFamily, protocol string, sourceIPs string, destinationIPs string, sourcePorts string, destinationPort string) *exec.Cmd {
	iptables := iptables(family)
	ipTablesCmd := exec.Command(iptables,
		"-t",
		"mangle",
		string(act),
		"PREROUTING",
		"-p",
		protocol,
		"--dport",
		destinationPort,
		"--match",
		"multiport",
		"--sports",
		sourcePorts,
		"-s",
		sourceIPs,
		"-d",
		destinationIPs,
		"-j",
		"NFQUEUE",
		"--queue-num",
		strconv.Itoa(nfq.nfqueueNumber))
	return ipTablesCmd
}

func (nfq *NFQueue) setProtocols(protocols []string) {
	nfq.protocols = []string{}
	for _, protocol := range protocols {
		if !validProtocol(protocol) {
			continue
		}
		nfq.protocols = append(nfq.protocols, protocol)
	}
}

func (nfq *NFQueue) setSourceIPs(sourceIPs []string) {
	nfq.sourceIPv4s = getValidIPs(ipv4, sourceIPs)
	nfq.sourceIPv6s = getValidIPs(ipv6, sourceIPs)
}

func (nfq *NFQueue) setDestinationIPs(destinationIPs []string) {
	nfq.destinationIPv4s = getValidIPs(ipv4, destinationIPs)
	nfq.destinationIPv6s = getValidIPs(ipv6, destinationIPs)
}

func (nfq *NFQueue) setSourcePorts(sourcePorts []string) {
	nfq.sourcePorts = getValidPorts(sourcePorts)
}

func (nfq *NFQueue) setDestinationPorts(destinationPorts []string) {
	nfq.destinationPorts = getValidPorts(destinationPorts)
}

func validProtocol(protocol string) bool {
	p := strings.ToLower(protocol)
	return p == "tcp" || p == "udp"
}

func formatPorts(ports []string) string {
	return strings.Join(ports, listSeperatorIpTables)
}

func getValidPorts(ports []string) []string {
	res := []string{}
	for _, port := range ports {
		_, err := strconv.Atoi(port)
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
		res = append(res, strings.Join(portRange, portRangeSeperatorIpTables))
	}
	return res
}

func formatIPs(ips []string) string {
	return strings.Join(ips, listSeperatorIpTables)
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

func getIPFamily(ip net.IP) ipFamily {
	if ip.To4() == nil {
		return ipv6
	}
	return ipv4
}

func iptables(family ipFamily) string {
	if family == ipv4 {
		return "iptables"
	}
	return "ip6tables"
}
