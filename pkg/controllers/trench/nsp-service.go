/*
Copyright (c) 2021-2022 Nordix Foundation
Copyright (c) 2025 OpenInfra Foundation Europe

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

	meridiov1 "github.com/nordix/meridio/api/v1"
	common "github.com/nordix/meridio/pkg/controllers/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NspService struct {
	trench *meridiov1.Trench
	model  *corev1.Service
	exec   *common.Executor
}

func NewNspService(e *common.Executor, t *meridiov1.Trench) (*NspService, error) {
	l := &NspService{
		trench: t.DeepCopy(),
		exec:   e,
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *NspService) getPorts() []corev1.ServicePort {
	// if nsp service ports are set in the cr, use the values
	// else return default service ports
	return []corev1.ServicePort{
		{
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(common.NspTargetPort),
			Port:       common.NspPort,
		},
	}
}

func (i *NspService) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.NSPServiceName(i.trench),
	}
}

func (i *NspService) insertParameters(svc *corev1.Service) *corev1.Service {
	// if status nsp service parameters are specified in the cr, use those
	// else use the default parameters
	ret := svc.DeepCopy()
	ret.ObjectMeta.Name = common.NSPServiceName(i.trench)
	ret.Spec.Selector["app"] = common.NSPStatefulSetName(i.trench)
	ret.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	ret.Spec.Ports = i.getPorts()
	return ret
}

func (i *NspService) getCurrentStatus() (*corev1.Service, error) {
	currentStatus := &corev1.Service{}
	selector := i.getSelector()
	err := i.exec.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get service object (%s): %w", selector.String(), err)
	}
	return currentStatus, nil
}

func (i *NspService) getDesiredStatus() *corev1.Service {
	return i.insertParameters(i.model)
}

// getReconciledDesiredStatus gets the desired status of nsp service after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NspService) getReconciledDesiredStatus(svc *corev1.Service) *corev1.Service {
	template := svc.DeepCopy()
	template.Spec.Type = i.model.Spec.Type
	template.ObjectMeta.Labels = common.MergeMapsInPlace(template.ObjectMeta.Labels, i.model.ObjectMeta.Labels)
	template.ObjectMeta.Annotations = common.MergeMapsInPlace(template.ObjectMeta.Annotations, i.model.ObjectMeta.Annotations)
	return i.insertParameters(template)
}

func (i *NspService) getModel() error {
	model, err := common.GetServiceModel("deployment/nsp-service.yaml")
	if err != nil {
		return fmt.Errorf("failed to get service model in deployment/nsp-service.yaml: %w", err)
	}
	i.model = model
	return nil
}

func (i *NspService) getAction() error {
	cs, err := i.getCurrentStatus()
	if err != nil {
		return err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		i.exec.AddCreateAction(ds)
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			i.exec.AddUpdateAction(ds)
		}
	}
	return nil
}
