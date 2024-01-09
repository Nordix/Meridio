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

package neighbor

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/nordix/meridio/pkg/log"
	"github.com/vishvananda/netlink"
)

var neighborStateToName = map[int]string{
	netlink.NUD_NONE:       "NONE",
	netlink.NUD_INCOMPLETE: "INCOMPLETE",
	netlink.NUD_REACHABLE:  "REACHABLE",
	netlink.NUD_STALE:      "STALE",
	netlink.NUD_DELAY:      "DELAY",
	netlink.NUD_PROBE:      "PROBE",
	netlink.NUD_FAILED:     "FAILED",
	netlink.NUD_NOARP:      "NOARP",
	netlink.NUD_PERMANENT:  "PERMANENT",
}

// NeighborReachabilityDetector -
// NeighborReachabilityDetector keeps track of certain neighbor entries for
// registered IPs to determine if neighbor address resolution is successful.
// If there is a change in reachability it triggers a logging event.
// In order to receive neighbor updates NeighborReachabilityDetector relies on
// NeighborMonitor.
// Note: Only neighbor updates with states REACHABLE and FAILED are processed.
type NeighborReachabilityDetector struct {
	cache           map[string]*NeighborCacheEntry
	name            string
	neighborMonitor *NeighborMonitor
	logger          logr.Logger
	mu              sync.Mutex
}

type NeighborCacheEntry struct {
	ip     net.IP    // for easy comparison with netlink.Neigh.IP
	ts     time.Time // timestamp of the most recent change
	tsLast time.Time // timestamp of the last "registered" occurence of the current update (for rate limiting)
	neigh  *netlink.Neigh
}

// Register -
// Registers IP addresses for which neighbor updates should be kept track of.
// Accepts IPs of both 192.0.2.1/24 and 192.0.2.1 format (but strips mask).
func (nrd *NeighborReachabilityDetector) Register(ips ...string) {
	nrd.logger.V(1).Info("Register", "IPs", ips)
	for _, ip := range ips {
		ipParts := strings.Split(ip, "/")
		ip := ipParts[0]
		netIP := net.ParseIP(ip)
		if netIP == nil {
			continue
		}
		nrd.mu.Lock()
		if _, ok := nrd.cache[ip]; !ok {
			nrd.cache[ip] = &NeighborCacheEntry{ip: netIP}
		}
		nrd.mu.Unlock()
	}
}

// Unregister -
func (nrd *NeighborReachabilityDetector) Unregister(ips ...string) {
	nrd.logger.V(1).Info("Unregister", "IPs", ips, "cacheLen", len(nrd.cache))
	for _, ip := range ips {
		ipParts := strings.Split(ip, "/")
		ip := ipParts[0]
		nrd.mu.Lock()
		delete(nrd.cache, ip)
		nrd.mu.Unlock()
	}
}

// Close -
// Close unsubscribes the detector from the neighbor monitor.
func (nrd *NeighborReachabilityDetector) Close() {
	if nrd.neighborMonitor != nil {
		nrd.neighborMonitor.UnSubscribe(nrd)
		nrd.logger.V(1).Info("Closed neighbor reachability detector")
	}
}

// NeighborUpdated -
// Process neighbor updates of interest referring to stored IPs.
// Checks if neighbor reachability changes (incuding MAC address update), and
// logs such events. Only FAILED <-> REACHABLE transitions are logged to keep
// track of MAC address resolution outcome.
//
// Note: Must NOT spam logs with reachable printouts in case of intermittent
// traffic causing recurring neighbor entry timeouts etc.
func (nrd *NeighborReachabilityDetector) NeighborUpdated(neighborUpdate netlink.NeighUpdate) {
	flags := netlink.NUD_FAILED | netlink.NUD_REACHABLE
	if neighborUpdate.State&flags == 0 {
		// update is neither FAILED nor REACHABLE
		return
	}

	nrd.mu.Lock()
	defer nrd.mu.Unlock()
	for ip, cacheEntry := range nrd.cache {
		if !cacheEntry.ip.Equal(neighborUpdate.IP) {
			continue
		}
		neigh := &neighborUpdate.Neigh
		// store neighbor update in case:
		// - no Neigh entry in cache yet
		// - FAILED <-> REACHABLE transition based on the last stored update
		// - MAC address is updated
		if cacheEntry.neigh == nil ||
			(cacheEntry.neigh.State|neigh.State)&flags == flags ||
			!bytes.Equal(cacheEntry.neigh.HardwareAddr, neigh.HardwareAddr) {
			cacheEntry.neigh = neigh
			cacheEntry.ts = time.Now()
			cacheEntry.tsLast = cacheEntry.ts
			logger := nrd.logger.WithValues(
				"IP", ip,
				"state", neighborStateToName[cacheEntry.neigh.State])
			if len(neigh.HardwareAddr) > 0 {
				logger = logger.WithValues("MAC", cacheEntry.neigh.HardwareAddr.String())
			}
			logger.V(1).Info("neighbor update", "neigh", neighborUpdate.Neigh)
			break
		}
		// long lasting neighbor resolution failure, allow one additional log
		//  printout per hour
		if cacheEntry.neigh != nil &&
			(cacheEntry.neigh.State&neigh.State)&netlink.NUD_FAILED != 0 &&
			cacheEntry.tsLast.Add(time.Hour).Before(time.Now()) {
			cacheEntry.tsLast = time.Now() // update timestamp
			nrd.logger.V(1).Info("neighbor update",
				"IP", ip,
				"state", neighborStateToName[cacheEntry.neigh.State],
				"since", cacheEntry.ts,
				"neigh", neighborUpdate.Neigh)
		}
	}
}

// NewNeighborReachabilityDetector -
// NewNeighborReachabilityDetector creates a new NeighborReachabilityDetector
// and registers it at the neighbor monitor.
func NewNeighborReachabilityDetector(ctx context.Context, name string, neighborMonitor *NeighborMonitor) (*NeighborReachabilityDetector, error) {
	if neighborMonitor == nil {
		return nil, fmt.Errorf("missing neighbor monitor")
	}

	neighborReachabilityDetector := &NeighborReachabilityDetector{
		name:            name,
		cache:           make(map[string]*NeighborCacheEntry),
		neighborMonitor: neighborMonitor,
		logger: log.FromContextOrGlobal(ctx).WithValues(
			"name", name,
			"class", "NeighborReachabilityDetector",
		),
	}
	neighborMonitor.Subscribe(neighborReachabilityDetector)

	neighborReachabilityDetector.logger.V(1).Info("Created neighbor reachability detector")
	return neighborReachabilityDetector, nil
}
