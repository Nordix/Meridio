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

package ipam

import (
	"context"
	"fmt"
	"net"

	"github.com/nordix/meridio/pkg/ipam/storage/memory"
)

type Ipam struct {
	Store Storage
}

func New() *Ipam {
	ipam := &Ipam{
		Store: memory.NewStorage(),
	}
	return ipam
}

func NewWithStorage(store Storage) *Ipam {
	ipam := &Ipam{
		Store: store,
	}
	return ipam
}

func (ipam *Ipam) AllocateSubnet(ctx context.Context, subnetPool string, prefixLength int) (string, error) {
	p, err := NewPrefixWithStorage(subnetPool, ipam.Store)
	if err != nil {
		return "", err
	}
	newSubnet, err := p.Allocate(ctx, prefixLength)
	if err != nil {
		return "", err
	}
	return newSubnet, nil
}

func (ipam *Ipam) ReleaseSubnet(ctx context.Context, subnetPool string, subnet string) error {
	p, err := NewPrefixWithStorage(subnetPool, ipam.Store)
	if err != nil {
		return err
	}
	err = p.Release(ctx, subnet)
	return err
}

func (ipam *Ipam) AllocateIP(ctx context.Context, subnet string) (string, error) {
	p, err := NewPrefixWithStorage(subnet, ipam.Store)
	if err != nil {
		return "", err
	}
	_, ipNet, _ := net.ParseCIDR(subnet) // should not return error since NewPrefixWithStorage
	lastIP := LastIP(ipNet)
	length, ipLength := ipNet.Mask.Size()
	var new net.IP
	for {
		newIP, err := p.Allocate(ctx, ipLength)
		new, _, _ = net.ParseCIDR(newIP) // should not return error since returned by Allocate
		if err != nil {
			return "", err
		}
		if new.String() != ipNet.IP.String() && new.String() != lastIP.String() {
			break
		}
	}
	return fmt.Sprintf("%s/%d", new.String(), length), nil
}

func (ipam *Ipam) ReleaseIP(ctx context.Context, subnet string, ip string) error {
	p, err := NewPrefixWithStorage(subnet, ipam.Store)
	if err != nil {
		return err
	}
	_, ipNet, _ := net.ParseCIDR(subnet) // should not return error since NewPrefixWithStorage
	_, ipLength := ipNet.Mask.Size()
	newIp, _, err := net.ParseCIDR(ip)
	if err != nil {
		return err
	}
	err = p.Release(ctx, fmt.Sprintf("%s/%d", newIp, ipLength))
	return err
}
