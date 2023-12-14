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

package flow

import (
	"fmt"

	"github.com/go-logr/logr"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/log"
)

// Flow holds flow data
type Flow struct {
	*nspAPI.Flow
	nfqueueLoadBalancer types.NFQueueLoadBalancer
	logger              logr.Logger
}

// New creates a new flow
func New(flow *nspAPI.Flow, lb types.NFQueueLoadBalancer) (types.Flow, error) {
	if flow == nil {
		return nil, fmt.Errorf("flow is nil")
	}
	if lb == nil {
		return nil, fmt.Errorf("missing nfqueue lb for flow (%s)", flow.Name)
	}
	logger := log.Logger.WithValues("class", "Flow",
		"instance", flow.Name,
		"nfqlb", lb.GetName(),
	)
	logger.Info("Create flow")
	f := &Flow{
		Flow:                flow,
		nfqueueLoadBalancer: lb,
		logger:              logger,
	}
	if err := f.nfqueueLoadBalancer.SetFlow(f.Flow); err != nil {
		return nil, fmt.Errorf("failed to set new flow (%s): %w", f.Flow.Name, err)
	}
	return f, nil
}

// Update updates a flow
func (f *Flow) Update(flow *nspAPI.Flow) error {
	if f.Flow.DeepEquals(flow) {
		// Not changed
		return nil
	}
	f.logger.V(2).Info("Update flow", "flow", flow)
	if f.Flow.Name != flow.Name {
		f.logger.V(1).Info("Attempted to update flow name", "name", f.Flow.Name, "to", flow.Name)
		return fmt.Errorf("flow name update is not allowed (%s -> %s)", f.Flow.Name, flow.Name)
	}

	f.Flow = flow
	err := f.nfqueueLoadBalancer.SetFlow(f.Flow)
	if err != nil {
		return fmt.Errorf("failed to update flow (%s): %w", flow.Name, err)
	}
	return nil
}

// Delete deletes a flow
func (f *Flow) Delete() error {
	f.logger.Info("Delete flow")
	err := f.nfqueueLoadBalancer.DeleteFlow(f.Flow)
	if err != nil {
		return fmt.Errorf("failed to delete flow (%s): %w", f.Flow.Name, err)
	}
	return nil
}
