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
	Replicas *int32 `json:"replicas,omitempty"`
	// vlan interface, cannot be updated
	VlanInterface string `json:"vlan-interface"`
	// vlan ID, cannot be updated
	VlanID int `json:"vlan-id"`
	// vlan ipv4 prefix
	VlanPrefixIPv4 string `json:"vlan-ipv4-prefix"`
	// vlan ipv6 prefix
	VlanPrefixIPv6 string `json:"vlan-ipv6-prefix"`
	// gateways that attractor expect to use
	Gateways []string `json:"gateways,omitempty"`
	// vips that attractor expect to use
	Vips []string `json:"vips,omitempty"`
}

// AttractorStatus defines the observed state of Attractor
type AttractorStatus struct {
	Message      string   `json:"message,omitempty"`
	LbFe         string   `json:"lb-fe,omitempty"`
	GatewayInUse []string `json:"gateways-in-use,omitempty"`
	VipsInUse    []string `json:"vips-in-use,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="VlanID",type=integer,JSONPath=`.spec.vlan-id`
//+kubebuilder:printcolumn:name="VlanITF",type=string,JSONPath=`.spec.vlan-interface`
//+kubebuilder:printcolumn:name="Gateways",type=string,JSONPath=`.spec.gateways`
//+kubebuilder:printcolumn:name="gw-in-use",type=string,JSONPath=`.status.gateways-in-use`
//+kubebuilder:printcolumn:name="Vips",type=string,JSONPath=`.spec.vips`
//+kubebuilder:printcolumn:name="vips-in-use",type=string,JSONPath=`.status.vips-in-use`
//+kubebuilder:printcolumn:name="trench",type=string,JSONPath=`.metadata.labels.trench`
//+kubebuilder:printcolumn:name="LB-FE",type=string,JSONPath=`.status.lb-fe`
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
