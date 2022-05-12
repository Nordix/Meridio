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

package proxy

import (
	"net"

	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
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
	return vip.sourceBasedRoute.AddNexthop(ip)
}

func (vip *virtualIP) RemoveNexthop(ip string) error {
	if !isSameFamily(vip.prefix, ip) {
		return nil
	}
	return vip.sourceBasedRoute.RemoveNexthop(ip)
}

func (vip *virtualIP) createSourceBaseRoute(tableID int) error {
	var err error
	vip.sourceBasedRoute, err = vip.netUtils.NewSourceBasedRoute(tableID, vip.prefix)
	logrus.Infof("Proxy: sourceBasedRoute index - vip: %v - %v", tableID, vip.prefix)
	if err != nil {
		return err
	}
	return nil
}

func (vip *virtualIP) removeSourceBaseRoute() error {
	return vip.sourceBasedRoute.Delete()
}

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
