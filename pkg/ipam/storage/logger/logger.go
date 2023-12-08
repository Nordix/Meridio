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

//nolint:wrapcheck
package logger

import (
	"context"

	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/nordix/meridio/pkg/log"
)

type Store struct {
	Store types.Storage
}

func (s *Store) Add(ctx context.Context, prefix types.Prefix) error {
	err := s.Store.Add(ctx, prefix)
	if err != nil {
		if prefix != nil {
			log.FromContextOrGlobal(ctx).Error(err, "Add", "Name", prefix.GetName(), "Cidr", prefix.GetCidr())
		}
		return err
	}
	if prefix != nil {
		log.FromContextOrGlobal(ctx).Info("Add", "Name", prefix.GetName(), "Cidr", prefix.GetCidr())
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, prefix types.Prefix) error {
	err := s.Store.Delete(ctx, prefix)
	if err != nil {
		if prefix != nil {
			log.FromContextOrGlobal(ctx).Error(err, "Delete", "Name", prefix.GetName(), "Cidr", prefix.GetCidr())
		}
		return err
	}
	if prefix != nil {
		log.FromContextOrGlobal(ctx).Info("Delete", "Name", prefix.GetName(), "Cidr", prefix.GetCidr())
	}
	return nil
}

func (s *Store) Get(ctx context.Context, name string, parent types.Prefix) (types.Prefix, error) {
	return s.Store.Get(ctx, name, parent)
}

func (s *Store) GetChilds(ctx context.Context, prefix types.Prefix) ([]types.Prefix, error) {
	return s.Store.GetChilds(ctx, prefix)
}
