/*
Copyright (c) 2021-2022 Nordix Foundation

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

package types

import (
	"context"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

type NFQueueLoadBalancer interface {
	Activate(index int, identifier int) error
	Deactivate(index int) error
	Start() error
	Delete() error
	SetFlow(flow *nspAPI.Flow) error
	DeleteFlow(flow *nspAPI.Flow) error
}

type NFQueueLoadBalancerFactory interface {
	Start(ctx context.Context) context.Context
	New(name string, m int, n int) (NFQueueLoadBalancer, error)
}

type NFAdaptor interface {
	SetDestinationIPs(vips []*nspAPI.Vip) error
}
