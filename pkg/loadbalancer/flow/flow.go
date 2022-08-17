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

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/sirupsen/logrus"
)

// Flow holds flow data
type Flow struct {
	*nspAPI.Flow
	nfqueueLoadBalancer types.NFQueueLoadBalancer
}

// New creates a new flow
func New(flow *nspAPI.Flow, lb types.NFQueueLoadBalancer) (types.Flow, error) {
	if flow == nil {
		return nil, fmt.Errorf("Flow:New: Create nil Flow")
	}
	logrus.Infof("Flow: New %s", flow.Name)
	if lb == nil {
		return nil, fmt.Errorf("Flow(%s):New: No NFQueueLoadBalancer", flow.Name)
	}
	f := &Flow{
		Flow:                flow,
		nfqueueLoadBalancer: lb,
	}
	if err := f.nfqueueLoadBalancer.SetFlow(f.Flow); err != nil {
		return nil, err
	}
	return f, nil
}

// Update updates a flow
func (f *Flow) Update(flow *nspAPI.Flow) error {
	logrus.Tracef("Flow:Update: %v -> %v", f.Flow, flow)
	if f.Flow.DeepEquals(flow) {
		// Not changed
		logrus.Debugf("Flow:Update: to same")
		return nil
	}
	if f.Flow.Name != flow.Name {
		logrus.Warningf("Flow:Update: name %v -> %v", f.Flow.Name, flow.Name)
		return fmt.Errorf("Flow:Update Name is not allowed")
	}

	f.Flow = flow
	return f.nfqueueLoadBalancer.SetFlow(f.Flow)
}

// Delete deletes a flow
func (f *Flow) Delete() error {
	logrus.Infof("Flow:Delete: %v", f.Flow)
	return f.nfqueueLoadBalancer.DeleteFlow(f.Flow)
}
