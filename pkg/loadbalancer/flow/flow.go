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
	"strconv"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/sirupsen/logrus"
)

// Flow holds flow data
type Flow struct {
	*nspAPI.Flow
	nfqueueLoadBalancer types.NFQueueLoadBalancer
	nftHandler types.NftHandler
}

// New creates a new flow
func New(flow *nspAPI.Flow, lb types.NFQueueLoadBalancer, nfth types.NftHandler) (types.Flow, error) {
	if flow == nil {
		return nil, fmt.Errorf("Flow:New: Create nil Flow")
	}
	logrus.Infof("Flow: New %s", flow.Name)
	if lb == nil {
		return nil, fmt.Errorf("Flow(%s):New: No NFQueueLoadBalancer", flow.Name)
	}
	if nfth == nil {
		return nil, fmt.Errorf("Flow(%s):New: No NftHandler", flow.Name)
	}

	if flow.LocalPort != 0 {
		dport, err := ensureSingleDport(flow)
		if err != nil {
			return nil, err
		}
		logrus.Infof("Flow:New: Port NAT: %v -> %v", dport, flow.LocalPort)

		err = nfth.PortNATCreateSets(flow)
		if err != nil {
			return nil, err
		}
		err = nfth.PortNATSetAddresses(flow)
		if err != nil {
			nfth.PortNATDeleteSets(flow)
			return nil, err
		}

		err = nfth.PortNATSet(
			flow.Name, flow.Protocols, dport, uint(flow.LocalPort))
		if err != nil {
			nfth.PortNATDeleteSets(flow)
			return nil, err
		}
	}

	f := &Flow{
		Flow:                flow,
		nfqueueLoadBalancer: lb,
		nftHandler:          nfth,
	}
	if err := f.nfqueueLoadBalancer.SetFlow(f.Flow); err != nil {
		if flow.LocalPort != 0 {
			nfth.PortNATDelete(flow.Name)
			nfth.PortNATDeleteSets(flow)
		}
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

	// Check port-NAT
	if flow.LocalPort != 0 {
		dport, err := ensureSingleDport(flow)
		if err != nil {
			return err
		}
		oldDport, _ := ensureSingleDport(f.Flow)

		if flow.LocalPort != f.Flow.LocalPort || dport != oldDport {
			logrus.Infof("Flow:Update: Port NAT: %v -> %v", dport, flow.LocalPort)
			if (f.Flow.LocalPort == 0) {
				err = f.nftHandler.PortNATCreateSets(flow)
				if err != nil {
					return err
				}
			}
			err = f.nftHandler.PortNATSet(
				flow.Name, flow.Protocols, dport, uint(flow.LocalPort))
			if err != nil {
				return err
			}
		}
		err = f.nftHandler.PortNATSetAddresses(flow)
		if err != nil {
			return err
		}

	} else if f.Flow.LocalPort != 0 {
		logrus.Infof("Flow:Update: Remove port NAT to %v",f.Flow.LocalPort)
		f.nftHandler.PortNATDelete(f.Flow.Name)
		f.nftHandler.PortNATDeleteSets(flow)
	}

	f.Flow = flow
	return f.nfqueueLoadBalancer.SetFlow(f.Flow)
}

// Delete deletes a flow
func (f *Flow) Delete() error {
	logrus.Infof("Flow:Delete: %v", f.Flow)
	if f.LocalPort != 0 {
		logrus.Infof("Flow:Delete port-NAT to %v", f.LocalPort)
		f.nftHandler.PortNATDelete(f.Flow.Name)
		f.nftHandler.PortNATDeleteSets(f.Flow)
	}
	return f.nfqueueLoadBalancer.DeleteFlow(f.Flow)
}

// Verify that only one dport is specified
func ensureSingleDport(flow *nspAPI.Flow) (uint, error) {
	if len(flow.DestinationPortRanges) != 1 {
		return 0, fmt.Errorf(
			"Flow(%s): Must have exactly one dport for port-NAT", flow.Name)
	}
	dport, err := strconv.Atoi(flow.DestinationPortRanges[0])
	if err != nil {
		return 0, fmt.Errorf(
			"Flow(%s): Dport range when using port-NAT", flow.Name)
	}
	return uint(dport), nil
}
