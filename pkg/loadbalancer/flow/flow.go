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
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/sirupsen/logrus"
)

type Flow struct {
	*nspAPI.Flow
	nfqueueLoadBalancer types.NFQueueLoadBalancer
}

func New(flow *nspAPI.Flow, options ...Option) (types.Flow, error) {
	opts := &flowOptions{}
	for _, opt := range options {
		opt(opts)
	}

	f := &Flow{
		Flow:                flow,
		nfqueueLoadBalancer: opts.nfqueueLoadBalancer,
	}

	var err error
	if f.nfqueueLoadBalancer != nil {
		err = f.nfqueueLoadBalancer.SetFlow(flow)
	}
	logrus.Infof("New flow: %v - err: %v", flow, err)
	return f, nil
}

func (f *Flow) Update(flow *nspAPI.Flow) error {
	if f.Flow == nil || !f.Flow.DeepEquals(flow) {
		f.Flow = flow
		logrus.Infof("Update flow: %v", f.Flow)
		if f.nfqueueLoadBalancer != nil {
			return f.nfqueueLoadBalancer.SetFlow(f.Flow)
		}
	}
	return nil
}

func (f *Flow) Delete() error {
	logrus.Infof("Delete flow: %v", f.Flow)
	if f.nfqueueLoadBalancer != nil {
		return f.nfqueueLoadBalancer.DeleteFlow(f.Flow)
	}
	return nil
}
