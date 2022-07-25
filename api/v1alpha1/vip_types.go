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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// VipSpec defines the desired state of Vip
type VipSpec struct {
	// vip address. Both ipv4 and ipv6 addresses are supported. Vip address must
	// be a valid network prefix.
	// +optional
	Address string `json:"address,omitempty"`
}

// VipStatus defines the observed state of Vip
type VipStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.spec.address`
//+kubebuilder:printcolumn:name="Trench",type=string,JSONPath=`.metadata.labels.trench`

// Vip is the Schema for the vips API. It defines a Vip to be used as destination
// for ingress traffic to facilitate load-balancing. For egress traffic the Vip
// is used as source to avoid Direct Server Return.
type Vip struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VipSpec   `json:"spec,omitempty"`
	Status VipStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VipList contains a list of Vip
type VipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Vip `json:"items"`
}

func (r *Vip) GroupResource() schema.GroupResource {
	return schema.GroupResource{
		Group:    r.GroupVersionKind().Group,
		Resource: r.GroupVersionKind().Kind,
	}
}

func (r *Vip) GroupKind() schema.GroupKind {
	return schema.GroupKind{
		Group: r.GroupVersionKind().Group,
		Kind:  r.GroupVersionKind().Kind,
	}
}

func init() {
	SchemeBuilder.Register(&Vip{}, &VipList{})
}
