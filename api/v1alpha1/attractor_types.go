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

// AttractorSpec defines the desired state of Attractor
type AttractorSpec struct {
	// replicas of attractor deployment
	LBReplicas  *int32 `json:"lb-replicas,omitempty"`
	NSEReplicas *int32 `json:"nse-replicas,omitempty"`
	// vlan interface, cannot be updated
	VlanInterface string `json:"vlan-interface,omitempty"`
	// vlan ID, cannot be updated
	VlanID int `json:"vlan-id"`
}

// AttractorStatus defines the observed state of Attractor
type AttractorStatus struct {
	Message string `json:"message,omitempty"`
	Vlan    string `json:"vlan,omitempty"`
	LB      string `json:"load-balancer,omitempty"`
	Status  string `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="LB-Replicas",type=integer,JSONPath=`.spec.lb-replicas`
//+kubebuilder:printcolumn:name="NSE-Replicas",type=integer,JSONPath=`.spec.nse-replicas`
//+kubebuilder:printcolumn:name="VlanID",type=integer,JSONPath=`.spec.vlan-id`
//+kubebuilder:printcolumn:name="VlanITF",type=string,JSONPath=`.spec.vlan-interface`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
//+kubebuilder:printcolumn:name="LB",type=string,JSONPath=`.status.load-balancer`
//+kubebuilder:printcolumn:name="VLAN",type=string,JSONPath=`.status.vlan`
// Attractor is the Schema for the attractors API
type Attractor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AttractorSpec   `json:"spec,omitempty"`
	Status AttractorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AttractorList contains a list of Attractor
type AttractorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Attractor `json:"items"`
}

func (r *Attractor) GroupResource() schema.GroupResource {
	return schema.GroupResource{
		Group:    r.GroupVersionKind().Group,
		Resource: r.GroupVersionKind().Kind,
	}
}

func init() {
	SchemeBuilder.Register(&Attractor{}, &AttractorList{})
}
