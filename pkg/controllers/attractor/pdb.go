/*
Copyright (c) 2022 Nordix Foundation

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

package attractor

import (
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LoadBalancerPDB struct {
	model      *policyv1.PodDisruptionBudget
	attractor  *meridiov1alpha1.Attractor
	exec       *common.Executor
	trench     *meridiov1alpha1.Trench
	pdbVersion string
}

func NewLoadBalancerPDB(e *common.Executor, attr *meridiov1alpha1.Attractor, t *meridiov1alpha1.Trench, pdbVersion string) (*LoadBalancerPDB, error) {
	pdb := &LoadBalancerPDB{
		attractor:  attr,
		exec:       e,
		trench:     t,
		pdbVersion: pdbVersion,
	}
	err := pdb.getModel()
	if err != nil {
		return nil, err
	}
	return pdb, nil
}

func (i *LoadBalancerPDB) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.PDBName(i.attractor),
	}
}

func (i *LoadBalancerPDB) insertParameters(pdb *policyv1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	ret := pdb.DeepCopy()
	ret.ObjectMeta.Name = common.PDBName(i.attractor)
	ret.ObjectMeta.Namespace = i.attractor.ObjectMeta.Namespace
	ret.Spec.Selector.MatchLabels["app"] = common.LbFeDeploymentName(i.attractor)
	return ret
}

func (i *LoadBalancerPDB) getModel() error {
	model, err := common.GetPodDisruptionBudgetModel("deployment/pdb.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *LoadBalancerPDB) getDesiredStatus() *policyv1.PodDisruptionBudget {
	return i.insertParameters(i.model)
}

func (i *LoadBalancerPDB) getReconciledDesiredStatus(pdb *policyv1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	template := pdb.DeepCopy()
	return i.insertParameters(template)
}

func (i *LoadBalancerPDB) getCurrentStatusV1() (*policyv1.PodDisruptionBudget, error) {
	currentStatus := &policyv1.PodDisruptionBudget{}
	selector := i.getSelector()
	err := i.exec.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentStatus, nil
}

func (i *LoadBalancerPDB) getCurrentStatusV1Beta1() (*policyv1beta1.PodDisruptionBudget, error) {
	currentStatus := &policyv1beta1.PodDisruptionBudget{}
	selector := i.getSelector()
	err := i.exec.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentStatus, nil
}

func (i *LoadBalancerPDB) getAction() error {
	if i.pdbVersion == policyv1beta1.SchemeGroupVersion.Version {
		return i.getActionV1Beta1()
	} else if i.pdbVersion == policyv1.SchemeGroupVersion.Version {
		return i.getActionV1()
	}
	return nil
}

func (i *LoadBalancerPDB) getActionV1() error {
	cs, err := i.getCurrentStatusV1()
	if err != nil {
		return err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		if err != nil {
			return err
		}
		i.exec.AddCreateAction(ds)
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds.Spec, cs.Spec) {
			i.exec.AddUpdateAction(ds)
		}
	}
	return nil
}

func (i *LoadBalancerPDB) getActionV1Beta1() error {
	csV1Beta1, err := i.getCurrentStatusV1Beta1()
	if err != nil {
		return err
	}
	cs := common.PdbV1Beta1ToV1(csV1Beta1)
	if cs == nil {
		ds := i.getDesiredStatus()
		if err != nil {
			return err
		}
		i.exec.AddCreateAction(common.PdbV1ToV1Beta1(ds))
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds.Spec, cs.Spec) {
			i.exec.AddUpdateAction(common.PdbV1ToV1Beta1(ds))
		}
	}
	return nil
}
