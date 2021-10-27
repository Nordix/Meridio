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

// FlowSpec defines the desired state of Flow
type FlowSpec struct {
	// Stream that this flow will sign up
	// +optional
	Stream string `json:"stream,omitempty"`

	// Vips that this flow will send traffic to
	// The vips shouldn't have overlapping
	Vips []string `json:"vips"`

	// Source subnets allowed in the flow
	// The subnets shouldn't have overlapping
	SourceSubnets []string `json:"source-subnets"`

	// Source port ranges allowed in the flow
	// The ports shouldn't have overlapping
	SourcePorts []string `json:"source-ports"`

	// Destination port ranges allowed in the flow
	// The ports shouldn't have overlapping
	DestinationPorts []string `json:"destination-ports"`

	// Protocols allowed in this flow
	// The protocols shouldn't have overlapping
	Protocols []string `json:"protocols"`
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
//+kubebuilder:printcolumn:name="stream",type=string,JSONPath=`.spec.stream`
//+kubebuilder:printcolumn:name="Trench",type=string,JSONPath=`.metadata.labels.trench`

// Flow is the Schema for the flows API
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
