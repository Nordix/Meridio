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

// FlowSpec defines the desired state of Flow
type FlowSpec struct {
	// Stream that is to include traffic classified by this flow
	// +optional
	Stream string `json:"stream,omitempty"`

	// Vips that this flow will send traffic to.
	// The vips should not have overlaps.
	Vips []string `json:"vips"`

	// Source subnets allowed in the flow.
	// The subnets should not have overlaps.
	SourceSubnets []string `json:"source-subnets,omitempty"`

	// Source port ranges allowed in the flow.
	// The ports should not have overlaps.
	// Ports can be defined by:
	// - a single port, such as 3000;
	// - a port range, such as 3000-4000;
	// - "any", which is equivalent to port range 0-65535.
	SourcePorts []string `json:"source-ports,omitempty"`

	// Destination port ranges allowed in the flow.
	// The ports should not have overlaps.
	// Ports can be defined by:
	// - a single port, such as 3000;
	// - a port range, such as 3000-4000;
	// - "any", which is equivalent to port range 0-65535.
	DestinationPorts []string `json:"destination-ports,omitempty"`

	// Protocols allowed in this flow.
	// The protocols should not have overlaps.
	Protocols []TransportProtocol `json:"protocols"`

	// Priority of the flow
	Priority int32 `json:"priority"`

	// ByteMatches matches bytes in the L4 header in the flow.
	// +optional
	ByteMatches []string `json:"byte-matches,omitempty"`
}

// FlowStatus defines the observed state of Flow
type FlowStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="vips",type=string,JSONPath=`.spec.vips`
//+kubebuilder:printcolumn:name="dst-ports",type=string,JSONPath=`.spec.destination-ports`
//+kubebuilder:printcolumn:name="src-subnets",type=string,JSONPath=`.spec.source-subnets`
//+kubebuilder:printcolumn:name="src-ports",type=string,JSONPath=`.spec.source-ports`
//+kubebuilder:printcolumn:name="protocols",type=string,JSONPath=`.spec.protocols`
//+kubebuilder:printcolumn:name="byte-matches",type=string,JSONPath=`.spec.byte-matches`
//+kubebuilder:printcolumn:name="stream",type=string,JSONPath=`.spec.stream`
//+kubebuilder:printcolumn:name="Trench",type=string,JSONPath=`.metadata.labels.trench`

// Flow is the Schema for the flows API. It defines how ingress
// traffic flows are classified and collected into streams
type Flow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlowSpec   `json:"spec,omitempty"`
	Status FlowStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FlowList contains a list of Flow
type FlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Flow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Flow{}, &FlowList{})
}

func (r *Flow) GroupKind() schema.GroupKind {
	return schema.GroupKind{
		Group: r.GroupVersionKind().Group,
		Kind:  r.GroupVersionKind().Kind,
	}
}

func (r *Flow) GroupResource() schema.GroupResource {
	return schema.GroupResource{
		Group:    r.GroupVersionKind().Group,
		Resource: r.GroupVersionKind().Kind,
	}
}
