/*
Copyright 2021.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VipSpec defines the desired state of Vip
type VipSpec struct {
	// vip address
	Address string `json:"address"`
}

// VipStatus defines the observed state of Vip
type VipStatus struct {
	// Describes if this vips is ready to be used
	// possible values:
	// - engaged: the vip is ready to be used
	// - disengaged: the vip cannot be used
	// - : vip is not processed by the controller yet
	Status ConfigStatus `json:"status,omitempty"`

	// Describes why Status is not "engaged"
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.spec.address`
//+kubebuilder:printcolumn:name="Trench",type=string,JSONPath=`.metadata.labels.trench`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`

// Vip is the Schema for the vips API
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

func init() {
	SchemeBuilder.Register(&Vip{}, &VipList{})
}
