/*
Copyright (c) 2024 OpenInfra Foundation Europe

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

package neighborcache

import (
	"context"
	"net"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/nordix/meridio/pkg/log"
	"github.com/vishvananda/netlink"
)

// RemoveInvalid attempts to remove potentially invalid neighbor entries for
// which NSM has reported that the connection was closed, implying that the
// interface has either disappeared or is about to disappear along with its
// IP and MAC addresses. Thus, even if the same IP address reappears shortly
// due to NSM heal successfully fixing or, more accurately, re-establishing the
// connection, communication disturbances caused by an old invalid neighbor
// cache entry can be avoided, which would have otherwise occurred due to the
// behavior of the neighbor state machine (DELAY state and unicast probes).
// Note: The LB monitors TAPA -> Proxy connections, where the SrcIpAddrs refer
// to TAPA-side IPs, including the ones used as Target IPs by the LB.
func RemoveInvalid(ctx context.Context, connectionEvent *networkservice.ConnectionEvent) {
	if connectionEvent.Type != networkservice.ConnectionEventType_DELETE {
		return
	}
	logger := log.FromContextOrGlobal(ctx).WithValues("func", "RemoveInvalid")
	// Fetch neighbor cache from kernel
	neighborList, err := netlink.NeighList(0, 0)
	if err != nil {
		logger.Info("Could not fetch neighbor list", "err", err)
		return
	}
	// Convert neighbor list to a map
	neighborMap := make(map[string][]netlink.Neigh)
	for _, neigh := range neighborList {
		ipStr := neigh.IP.String()
		neighborMap[ipStr] = append(neighborMap[ipStr], neigh)
	}

	// Remove any of the NSM SrcIpAddrs from the neighbor cache if they are present
	eventPrinted := false
	for _, connection := range connectionEvent.Connections {
		if connection.GetPath() == nil || len(connection.GetPath().GetPathSegments()) < 1 {
			continue
		}
		if connection.GetContext() == nil || connection.GetContext().GetIpContext() == nil {
			continue
		}
		ipContext := connection.GetContext().GetIpContext()
		for _, ipStr := range ipContext.SrcIpAddrs {
			if ip, _, err := net.ParseCIDR(ipStr); err == nil {
				// Check if neighbor map has an entry for this IP
				neighs, ok := neighborMap[ip.String()]
				if !ok {
					continue
				}
				if !eventPrinted {
					eventPrinted = true
					logger.Info("Connection event", "event", connectionEvent)
				}
				for _, neigh := range neighs {
					logger.Info("Delete from neighbor cache", "neigh", neigh, "MAC", neigh.HardwareAddr.String())
					err := netlink.NeighDel(&netlink.Neigh{
						LinkIndex: neigh.LinkIndex,
						IP:        ip,
					})
					if err != nil {
						logger.Info("Failed to delete from neighbor cache", "neigh", neigh, "MAC", neigh.HardwareAddr.String(), "err", err)
					}
				}
			}
		}
	}
}
