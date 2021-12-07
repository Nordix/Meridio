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

type Prefix struct {
	Cidr  string
	Store Storage
}

func NewPrefix(prefix string) (*Prefix, error) {
	if !IsCIDR(prefix) {
		return nil, fmt.Errorf("not a cidr: %v", prefix)
	}
	p := &Prefix{
		Cidr:  prefix,
		Store: memory.NewStorage(),
	}
	return p, nil
}

func NewPrefixWithStorage(prefix string, store Storage) (*Prefix, error) {
	if !IsCIDR(prefix) {
		return nil, fmt.Errorf("not a cidr: %v", prefix)
	}
	p := &Prefix{
		Cidr:  prefix,
		Store: store,
	}
	return p, nil
}

func (p *Prefix) Allocate(ctx context.Context, length int) (string, error) {
	if length <= p.GetLength() {
		return "", fmt.Errorf("invalid prefix length requested (should be > to %v): %v", p.GetLength(), length)
	}
	_, currentCandidate, err := net.ParseCIDR(fmt.Sprintf("%s/%d", p.GetAddress(), length))
	if err != nil {
		return "", err
	}
	firstCandidate := currentCandidate
	for {
		childs, err := p.getChilds(ctx)
		if err != nil {
			return "", err
		}
		for {
			collision := p.collideWith(currentCandidate.String(), childs)
			if collision == "" {
				break
			}
			_, collisionIPNet, _ := net.ParseCIDR(collision) // Jump over the collision
			_, currentCandidate, _ = net.ParseCIDR(fmt.Sprintf("%s/%d", LastIP(collisionIPNet).String(), length))
			currentCandidate = NextPrefix(currentCandidate)
			if firstCandidate.String() == currentCandidate.String() || !OverlappingPrefixes(p.Cidr, currentCandidate.String()) { // check if prefix contains candidate
				return "", fmt.Errorf("no more prefix available")
			}
		}
		err = p.Store.Add(ctx, p.Cidr, currentCandidate.String())
		if err == nil {
			break
		}
	}
	return currentCandidate.String(), nil
}

func (p *Prefix) Release(ctx context.Context, child string) error {
	if !IsCIDR(child) {
		return fmt.Errorf("not a cidr: %v", child)
	}
	err := p.Store.Delete(ctx, p.Cidr, child)
	return err
}

func (p *Prefix) getChilds(ctx context.Context) ([]string, error) {
	// todo: sort childs
	return p.Store.Get(ctx, p.Cidr)
}

func (p *Prefix) collideWith(prefix string, childs []string) string {
	for _, childPrefix := range childs {
		if OverlappingPrefixes(childPrefix, prefix) {
			return childPrefix
		}
	}
	return ""
}

func (p *Prefix) GetFamily() IPFamily {
	family, _ := GetFamily(p.Cidr)
	return family
}

func (p *Prefix) GetIPNet() *net.IPNet {
	_, ipnet, _ := net.ParseCIDR(p.Cidr)
	return ipnet
}

func (p *Prefix) GetAddress() string {
	_, ipnet, _ := net.ParseCIDR(p.Cidr)
	return ipnet.IP.String()
}

func (p *Prefix) GetLength() int {
	_, ipnet, _ := net.ParseCIDR(p.Cidr)
	length, _ := ipnet.Mask.Size()
	return length
}
