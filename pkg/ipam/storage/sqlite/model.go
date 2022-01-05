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

package sqlite

import (
	"fmt"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
)

type Prefix struct {
	Id       string `gorm:"primaryKey"`
	Name     string
	Cidr     string
	ParentID string
	Parent   *Prefix
}

func modelToPrefix(p *Prefix) types.Prefix {
	if p == nil {
		return nil
	}
	prefix := &prefix.Prefix{
		Name:   p.Name,
		Cidr:   p.Cidr,
		Parent: modelToPrefix(p.Parent),
	}
	return prefix
}

func modelToPrefixWithParent(p *Prefix, parent types.Prefix) types.Prefix {
	if p == nil {
		return nil
	}
	prefix := &prefix.Prefix{
		Name:   p.Name,
		Cidr:   p.Cidr,
		Parent: parent,
	}
	return prefix
}

func prefixToModel(p types.Prefix) *Prefix {
	if p == nil {
		return nil
	}
	parent := prefixToModel(p.GetParent())
	var parentID string
	id := p.GetName()
	if parent != nil {
		parentID = parent.Id
		id = fmt.Sprintf("%s-%s", id, parentID)
	}
	prefix := &Prefix{
		Id:       id,
		Name:     p.GetName(),
		Cidr:     p.GetCidr(),
		ParentID: parentID,
		Parent:   parent,
	}
	return prefix
}
