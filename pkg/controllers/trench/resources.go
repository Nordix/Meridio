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

package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio/api/v1alpha1"
	common "github.com/nordix/meridio/pkg/controllers/common"
)

type Resources interface {
	getAction() error
}

type Meridio struct {
	ipamStatefulSet *IpamStatefulSet
	ipamService     *IpamService
	nspStatefulSet  *NspStatefulSet
	nspService      *NspService
	configmap       *ConfigMap
}

func NewMeridio(e *common.Executor, trench *meridiov1alpha1.Trench) (*Meridio, error) {
	ipamsvc, err := NewIPAMSvc(e, trench)
	if err != nil {
		return nil, err
	}
	ipam, err := NewIPAM(e, trench)
	if err != nil {
		return nil, err
	}
	nspd, err := NewNspStatefulSet(e, trench)
	if err != nil {
		return nil, err
	}
	nsps, err := NewNspService(e, trench)
	if err != nil {
		return nil, err
	}
	cfg := NewConfigMap(e, trench)
	return &Meridio{
		ipamStatefulSet: ipam,
		ipamService:     ipamsvc,
		nspStatefulSet:  nspd,
		nspService:      nsps,
		configmap:       cfg,
	}, nil
}

func (m Meridio) ReconcileAll() error {
	resources := []Resources{
		m.nspStatefulSet,
		m.nspService,
		m.ipamStatefulSet,
		m.ipamService,
		m.configmap,
	}

	for _, r := range resources {
		err := r.getAction()
		if err != nil {
			return fmt.Errorf("get %t action error: %s", r, err)
		}
	}
	return nil
}
