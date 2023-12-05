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

package stream

import (
	"context"
	"fmt"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

type targetRegistryImpl struct {
	TargetRegistryClient nspAPI.TargetRegistryClient
}

func newTargetRegistryImpl(targetRegistryClient nspAPI.TargetRegistryClient) *targetRegistryImpl {
	tri := &targetRegistryImpl{
		TargetRegistryClient: targetRegistryClient,
	}
	return tri
}

func (tri *targetRegistryImpl) Register(ctx context.Context, target *nspAPI.Target) error {
	if _, err := tri.TargetRegistryClient.Register(ctx, target); err != nil {
		return fmt.Errorf("target registry register error: %w", err)
	}
	return nil
}

func (tri *targetRegistryImpl) Unregister(ctx context.Context, target *nspAPI.Target) error {
	if _, err := tri.TargetRegistryClient.Unregister(ctx, target); err != nil {
		return fmt.Errorf("target registry unregister error: %w", err)
	}
	return nil
}

func (tri *targetRegistryImpl) GetTargets(ctx context.Context, target *nspAPI.Target) ([]*nspAPI.Target, error) {
	watchClient, err := tri.TargetRegistryClient.Watch(ctx, target)
	if err != nil {
		return []*nspAPI.Target{}, fmt.Errorf("target registry failed to create watch client: %w", err)
	}
	responseTargets, err := watchClient.Recv()
	if err != nil {
		return []*nspAPI.Target{}, fmt.Errorf("target registry watch client receive error: %w", err)
	}
	return responseTargets.GetTargets(), nil
}
