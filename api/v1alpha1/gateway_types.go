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

// GatewaySpec defines the desired state of Gateway
type GatewaySpec struct {
	// Address of the Edge Router
	Address string `json:"address"`

	// +kubebuilder:default=bgp

	// The routing choice between the gateway and frontend
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Parameters to set up the BGP session to specified Address.
	// If the Protocol is static, this property must be empty.
	// If the Protocol is bgp, the minimal parameters to be defined in BgpSpec are RemoteASN and LocalASN.
	// +optional
	Bgp BgpSpec `json:"bgp,omitempty"`

	// Parameters to work with the static routing configured on the Edge Router with specified Address
	// If the Protocol is bgp, this property must be empty.
	// +optional
	Static StaticSpec `json:"static,omitempty"`
}

// BgpSpec defines the parameters to set up a BGP session
type BgpSpec struct {
	// The ASN number of the Gateway
	RemoteASN *uint32 `json:"remote-asn"`

	// The ASN number of the system where the FrontEnd locates
	LocalASN *uint32 `json:"local-asn"`

	// +kubebuilder:default=false

	// BFD monitoring of BGP session.
	// Valid values are:
	// - false (default): no BFD monitoring
	// - true: turns on the BFD monitoring. (Currently not supported)
	// +optional
	BFD *bool `json:"bfd,omitempty"`

	// +kubebuilder:default="240s"

	// Hold timer of the BGP session. Please refere to BGP material to understand what this implies.
	// The value must be a valid duration format. For example, 90s, 1m, 1h
	// Minimum duration is 3s. Default: 240s
	// +optional
	HoldTime string `json:"hold-time,omitempty"`

	// +kubebuilder:default=179

	// BGP listening port of the gateway. Default 179
	// +optional
	RemotePort *uint16 `json:"remote-port,omitempty"`

	// +kubebuilder:default=179

	// BGP listening port of the frontend. Default 179
	// +optional
	LocalPort *uint16 `json:"local-port,omitempty"`
}

// StaticSpec defines the parameters to set up static routes
type StaticSpec struct {
}

// GatewayStatus defines the observed state of Gateway
type GatewayStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="address",type=string,JSONPath=`.spec.address`
//+kubebuilder:printcolumn:name="protocol",type=string,JSONPath=`.spec.protocol`
//+kubebuilder:printcolumn:name="trench",type=string,JSONPath=`.metadata.labels.trench`

// Gateway is the Schema for the gateways API
type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewaySpec   `json:"spec,omitempty"`
	Status GatewayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GatewayList contains a list of Gateway
type GatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gateway `json:"items"`
}

func (r *Gateway) GroupResource() schema.GroupResource {
	return schema.GroupResource{
		Group:    r.GroupVersionKind().Group,
		Resource: r.GroupVersionKind().Kind,
	}
}

func (r *Gateway) GroupKind() schema.GroupKind {
	return schema.GroupKind{
		Group: r.GroupVersionKind().Group,
		Kind:  r.GroupVersionKind().Kind,
	}
}

func init() {
	SchemeBuilder.Register(&Gateway{}, &GatewayList{})
}
