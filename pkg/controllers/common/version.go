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

package common

import (
	"reflect"

	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetPodDisruptionBudgetVersion(client client.Client) (string, error) {
	t := reflect.TypeOf(&policyv1.PodDisruptionBudget{})
	gk := schema.GroupKind{
		Group: policyv1.GroupName,
		Kind:  t.Elem().Name(),
	}
	gvk, err := client.RESTMapper().RESTMapping(gk)
	if err != nil {
		return "", err
	}
	return gvk.GroupVersionKind.Version, nil
}

func PdbV1Beta1ToV1(pdb *policyv1beta1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	if pdb == nil {
		return nil
	}
	v1Pdb := &policyv1.PodDisruptionBudget{}
	v1Pdb.ObjectMeta = pdb.ObjectMeta
	v1Pdb.Spec = policyv1.PodDisruptionBudgetSpec(pdb.Spec)
	return v1Pdb
}

func PdbV1ToV1Beta1(pdb *policyv1.PodDisruptionBudget) *policyv1beta1.PodDisruptionBudget {
	if pdb == nil {
		return nil
	}
	v1beta1Pdb := &policyv1beta1.PodDisruptionBudget{}
	v1beta1Pdb.ObjectMeta = pdb.ObjectMeta
	v1beta1Pdb.Spec = policyv1beta1.PodDisruptionBudgetSpec(pdb.Spec)
	return v1beta1Pdb
}
