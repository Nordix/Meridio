/*
Copyright (c) 2024 Nordix Foundation

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

package debug

import (
	"encoding/json"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/vishvananda/netlink"
)

func Collect() *Export {
	export := &Export{
		Version:              Version,
		MeridioVersion:       MeridioVersion,
		UnixTime:             time.Now().Unix(),
		NetworkInterfaces:    listNetworkInterfaces(),
		Neighbors:            listNeighbors(),
		Routes:               listRoutes(),
		Rules:                listRules(),
		System:               getSystemInfo(),
		EnvironmentVariables: listEnvironmentVariables(),
	}

	return export
}

func (e *Export) String() string {
	res, _ := json.Marshal(e)

	return string(res)
}

func listNetworkInterfaces() []*NetworkInterface {
	networkInterfaces := []*NetworkInterface{}
	links, err := netlink.LinkList()
	if err != nil {
		return nil
	}

	for _, link := range links {
		ips := []string{}
		addresses, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			continue
		}
		for _, addr := range addresses {
			ips = append(ips, addr.IPNet.String())
		}

		networkInterfaces = append(networkInterfaces, &NetworkInterface{
			Index:       link.Attrs().Index,
			Name:        link.Attrs().Name,
			Mac:         link.Attrs().HardwareAddr.String(),
			IPs:         ips,
			MTU:         link.Attrs().MTU,
			Up:          link.Attrs().Flags&net.FlagUp == net.FlagUp,
			MasterIndex: link.Attrs().MasterIndex,
			Statistics: &Statistics{
				RxPackets: link.Attrs().Statistics.RxPackets,
				TxPackets: link.Attrs().Statistics.TxPackets,
				RxBytes:   link.Attrs().Statistics.RxBytes,
				TxBytes:   link.Attrs().Statistics.TxBytes,
				RxErrors:  link.Attrs().Statistics.RxErrors,
				TxErrors:  link.Attrs().Statistics.TxErrors,
				RxDropped: link.Attrs().Statistics.RxDropped,
				TxDropped: link.Attrs().Statistics.TxDropped,
			},
		})
	}

	return networkInterfaces
}

func listNeighbors() []*Neighbor {
	neighbors := []*Neighbor{}
	neighborList, err := netlink.NeighList(0, netlink.FAMILY_ALL)
	if err != nil {
		return nil
	}

	// https://github.com/vishvananda/netlink/blob/v1.0.0/neigh_linux.go#L24
	states := map[int]string{
		0x00: "none",
		0x01: "Incomplete",
		0x02: "Reachable",
		0x04: "Stale",
		0x08: "Delay",
		0x10: "Probe",
		0x20: "Failed",
		0x40: "No ARP",
		0x80: "Permanent",
	}

	for _, n := range neighborList {
		state, exists := states[n.State]
		if exists {
			state = strconv.Itoa(n.State)
		}
		neighbors = append(neighbors, &Neighbor{
			IP:             n.IP.String(),
			Mac:            n.HardwareAddr.String(),
			State:          state,
			InterfaceIndex: n.LinkIndex,
		})
	}

	return neighbors
}

func listRoutes() []*Route {
	routes := []*Route{}
	routeList, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return nil
	}

	for _, r := range routeList {
		nexthops := []string{}
		for _, nexthop := range r.MultiPath {
			if nexthop.Gw == nil {
				continue
			}
			nexthops = append(nexthops, nexthop.Gw.String())
		}

		destination := "default"
		if r.Dst != nil {
			destination = r.Dst.String()
		}

		gateway := ""
		if r.Gw != nil {
			gateway = r.Gw.String()
		}

		source := ""
		if r.Src != nil {
			source = r.Src.String()
		}

		routes = append(routes, &Route{
			Table:       r.Table,
			Destination: destination,
			Nexthops:    nexthops,
			Gateway:     gateway,
			Source:      source,
		})
	}

	return routes
}

func listRules() []*Rule {
	rules := []*Rule{}
	ruleList, err := netlink.RuleList(netlink.FAMILY_ALL)
	if err != nil {
		return nil
	}

	for _, r := range ruleList {
		source := ""
		if r.Dst != nil {
			source = r.Src.String()
		}

		destination := ""
		if r.Dst != nil {
			destination = r.Dst.String()
		}
		rules = append(rules, &Rule{
			Table:       r.Table,
			Priority:    r.Priority,
			Mark:        int(r.Mark), // TODO: revisit why it's an int and if changing to uint32 would be painful
			Source:      source,
			Destination: destination,
		})
	}

	return rules
}

func getSystemInfo() *System {
	s := &System{}

	cpuInfo, err := cpu.Info()
	if err == nil {
		s.CPUInfo = cpuInfo
	}

	hostInfo, err := host.Info()
	if err == nil {
		s.HostInfo = hostInfo
	}

	return s
}

func listEnvironmentVariables() []string {
	return os.Environ()
}
