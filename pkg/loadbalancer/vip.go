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

package loadbalancer

import (
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
)

type virtualIP struct {
	prefix   string
	netUtils networking.Utils
	nfqueue  networking.NFQueue
}

func (vip *virtualIP) Delete() error {
	return vip.nfqueue.Delete()
}

func (vip *virtualIP) createNFQueue() error {
	var err error
	vip.nfqueue, err = vip.netUtils.NewNFQueue(vip.prefix, 2)
	if err != nil {
		logrus.Errorf("Load Balancer: error configuring nfqueue (iptables): %v", err)
		return err
	}
	return nil
}

func newVirtualIP(prefix string, netUtils networking.Utils) (*virtualIP, error) {
	vip := &virtualIP{
		prefix:   prefix,
		netUtils: netUtils,
	}
	err := vip.createNFQueue()
	if err != nil {
		return nil, err
	}
	return vip, nil
}
