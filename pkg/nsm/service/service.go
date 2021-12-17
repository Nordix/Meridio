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

package service

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type Service struct {
	NSRegistryClient registry.NetworkServiceRegistryClient
	NS               *registry.NetworkService
}

func New(nsRegistryClient registry.NetworkServiceRegistryClient,
	ns *registry.NetworkService) *Service {
	s := &Service{
		NSRegistryClient: nsRegistryClient,
		NS:               ns,
	}
	return s
}

func (s *Service) Register(ctx context.Context) error {
	var err error
	s.NS, err = s.NSRegistryClient.Register(ctx, s.NS)
	return err
}

func (s *Service) Unregister(ctx context.Context) error {
	_, err := s.NSRegistryClient.Unregister(ctx, s.NS)
	return err
}
