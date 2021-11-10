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

package flow

import (
	"fmt"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
)

type Flow struct {
	*nspAPI.Flow
	NFQueue networking.NFQueue
}

func New(flow *nspAPI.Flow, nfqueueNumber int, nfqueueFactory networking.NFQueueFactory) (types.Flow, error) {
	f := &Flow{
		Flow: flow,
	}
	nfqueue, err := nfqueueFactory.NewNFQueue(
		f.getNFQueueName(),
		uint16(nfqueueNumber),
		f.GetProtocols(),
		f.GetSourceSubnets(),
		f.getVipAddresses(),
		f.GetSourcePortRanges(),
		f.GetDestinationPortRanges(),
		f.Priority)
	logrus.Infof("New flow: %v - %v", flow, err)
	if err != nil {
		return nil, err
	}
	f.NFQueue = nfqueue
	return f, nil
}

func (f *Flow) Update(flow *nspAPI.Flow) error {
	f.Flow = flow
	logrus.Infof("Update flow: %v", f.Flow)
	return f.NFQueue.Update(
		f.GetProtocols(),
		f.GetSourceSubnets(),
		f.getVipAddresses(),
		f.GetSourcePortRanges(),
		f.GetDestinationPortRanges())
}

func (f *Flow) Delete() error {
	logrus.Infof("Delete flow: %v", f.Flow)
	return f.NFQueue.Delete()
}

func (f *Flow) getVipAddresses() []string {
	vips := []string{}
	for _, vip := range f.GetVips() {
		vips = append(vips, vip.Address)
	}
	return vips
}

func (f *Flow) getNFQueueName() string {
	// Concatenate flow name + stream name to avoid nfqueue conflicts when
	// a flow is move from one stream to another (the receiving flow could
	// create the nfqueue before the old one is removed).
	return fmt.Sprintf("%s-%s", f.GetName(), f.GetStream().GetName())
}
