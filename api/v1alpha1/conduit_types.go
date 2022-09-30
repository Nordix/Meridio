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

// ConduitSpec defines the desired state of Conduit
type ConduitSpec struct {
	// +kubebuilder:default=stateless-lb
	// +kubebuilder:validation:Enum=stateless-lb
	// Type is the type of network service for this conduit
	Type string `json:"type"`

	// List of destination ports to NAT.
	DestinationPortNats []PortNatSpec `json:"destination-port-nats,omitempty"`
}

// ConduitStatus defines the observed state of Conduit
type ConduitStatus struct {
}

// PortNatSpec defines the parameters to set up a destination port natting in the conduit
type PortNatSpec struct {
	// Destination Port exposed by the service (exposed in flows).
	// Traffic containing this property will be NATted.
	Port uint16 `json:"port"`

	// TargetPort represent the port the traffic will be NATted to.
	// Targets will receive traffic on that port.
	TargetPort uint16 `json:"target-port"`

	// VIPs exposed by the service (exposed in flows).
	// Traffic containing this property will be NATted.
	Vips []string `json:"vips"`

	// Protocol exposed by the service (exposed in flows).
	// Traffic containing this property will be NATted.
	Protocol TransportProtocol `json:"protocol"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Trench",type=string,JSONPath=`.metadata.labels.trench`

// Conduit is the Schema for the conduits API. It defines a logical/physical
// traffic-path through the k8s cluster for processing traffic streams
type Conduit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConduitSpec   `json:"spec,omitempty"`
	Status ConduitStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConduitList contains a list of Conduit
type ConduitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Conduit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Conduit{}, &ConduitList{})
}

func (r *Conduit) GroupKind() schema.GroupKind {
	return schema.GroupKind{
		Group: r.GroupVersionKind().Group,
		Kind:  r.GroupVersionKind().Kind,
	}
}

func (r *Conduit) GroupResource() schema.GroupResource {
	return schema.GroupResource{
		Group:    r.GroupVersionKind().Group,
		Resource: r.GroupVersionKind().Kind,
	}
}
