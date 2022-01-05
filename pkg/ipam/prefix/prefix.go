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

package prefix

import (
	"github.com/nordix/meridio/pkg/ipam/types"
)

type Prefix struct {
	Name   string
	Cidr   string
	Parent types.Prefix
}

func New(name string, cidr string, parent types.Prefix) types.Prefix {
	p := &Prefix{
		Name:   name,
		Cidr:   cidr,
		Parent: parent,
	}
	return p
}

func (p *Prefix) GetName() string {
	return p.Name
}

func (p *Prefix) GetCidr() string {
	return p.Cidr
}

func (p *Prefix) GetParent() types.Prefix {
	return p.Parent
}

func (p *Prefix) Equals(prefix types.Prefix) bool {
	if prefix == nil {
		return false
	}
	parent := p.GetParent() == prefix.GetParent()
	if p.GetParent() != nil {
		parent = p.GetParent().Equals(prefix.GetParent())
	}
	return p.GetName() == prefix.GetName() && parent
}
