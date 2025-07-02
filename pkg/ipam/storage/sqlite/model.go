/*
Copyright (c) 2021 Nordix Foundation
Copyright (c) 2024-2025 OpenInfra Foundation Europe

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
	"errors"
	"fmt"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
	"gorm.io/gorm"
)

type Prefix struct {
	Id        string `gorm:"primaryKey"`
	Name      string `gorm:"index"`                          // supposedly indexing could improve query performance
	Cidr      string `gorm:"uniqueIndex:idx_parent_id_cidr"` // composite uniqueIndex should be helpful in resolving race conditions in concurrent allocation attempts considering the hierarchical allocation logic of prefix.Allocate
	ParentID  string `gorm:"index;uniqueIndex:idx_parent_id_cidr"`
	Parent    *Prefix
	UpdatedAt time.Time `gorm:"index"`               // supposedly indexing could improve query performance
	Expirable *bool     `gorm:"index;default:false"` // indicates whether prefix can expire and thus be subject to garbage collection
}

// isCIDRUniquenessViolation checks if error code reports unique index violation for CIDR
// Note: Error string is not matched as it would make the code brittle, however the check
// exploits the fact that the model contains only one (composite) unique index.
func isCIDRUniquenessViolation(err error) bool {
	if err == nil {
		return false
	}

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return true
		}
	}
	return false
}

func modelToPrefix(p *Prefix, parent types.Prefix) types.Prefix {
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

// TableNameForModel infers the table name for a given model
func TableNameForModel(db *gorm.DB, model any) string {
	stmt := &gorm.Statement{DB: db}
	_ = stmt.Parse(model)
	return stmt.Schema.Table
}
