/*
Copyright (c) 2021-2023 Nordix Foundation

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

package proxy

import (
	"errors"
	"fmt"
	"io/fs"
	"net"

	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/networking"
)

type virtualIP struct {
	sourceBasedRoute networking.SourceBasedRoute
	prefix           string
	netUtils         networking.Utils
}

func isSameFamily(cidr1, cidr2 string) bool {
	ip1, _, err := net.ParseCIDR(cidr1)
	if err != nil {
		return false
	}
	ip2, _, err := net.ParseCIDR(cidr2)
	if err != nil {
		return false
	}
	return (ip1.To4() == nil) == (ip2.To4() == nil)
}

func (vip *virtualIP) Delete() error {
	return vip.removeSourceBaseRoute()
}

func (vip *virtualIP) AddNexthop(ip string) error {
	if !isSameFamily(vip.prefix, ip) {
		return nil
	}
	if err := vip.sourceBasedRoute.AddNexthop(ip); err != nil {
		if errors.Is(err, fs.ErrExist) { // error return only needed to avoid spamming printouts "Added nexthop"
			return nil
		}
		return fmt.Errorf("error adding source route with nexthop %s for prefix %s: %w", ip, vip.prefix, err)
	}
	log.Logger.Info("Added nexthop", "nexthop", ip, "src prefix", vip.prefix)
	return nil
}

func (vip *virtualIP) RemoveNexthop(ip string) error {
	if !isSameFamily(vip.prefix, ip) {
		return nil
	}
	if err := vip.sourceBasedRoute.RemoveNexthop(ip); err != nil {
		return fmt.Errorf("error removing source route with nexthop %s for prefix %s: %w", ip, vip.prefix, err)
	}
	log.Logger.Info("Removed nexthop", "nexthop", ip, "src prefix", vip.prefix)
	return nil
}

func (vip *virtualIP) createSourceBaseRoute(tableID int) error {
	var err error
	vip.sourceBasedRoute, err = vip.netUtils.NewSourceBasedRoute(tableID, vip.prefix)
	if err != nil {
		return fmt.Errorf("error creating source policy routing rule for prefix %s tableID %v: %w", vip.prefix, tableID, err)
	}
	log.Logger.V(1).Info("Created source policy routing rule", "tableID", tableID, "prefix", vip.prefix)
	return nil
}

func (vip *virtualIP) removeSourceBaseRoute() error {
	if err := vip.sourceBasedRoute.Delete(); err != nil {
		return fmt.Errorf("error deleting source policy routing rule and routes for prefix %s: %w", vip.prefix, err)
	}
	log.Logger.V(1).Info("Removed source policy routing rule", "prefix", vip.prefix)
	return nil
}

// TODO: Avoid multiple source policy rules for the same VIP prefix (e.g. after container crash)
func newVirtualIP(prefix string, tableID int, netUtils networking.Utils) (*virtualIP, error) {
	vip := &virtualIP{
		prefix:   prefix,
		netUtils: netUtils,
	}
	err := vip.createSourceBaseRoute(tableID)
	if err != nil {
		return nil, err
	}
	return vip, nil
}
