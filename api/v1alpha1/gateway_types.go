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

// GatewaySpec defines the desired state of Gateway
type GatewaySpec struct {
	// Address of the Gateway Router
	Address string `json:"address"`

	// +kubebuilder:default=bgp
	// +kubebuilder:validation:Enum=bgp;static

	// The routing choice between the Gateway Router and Attractor FrontEnds.
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Parameters to set up the BGP session to specified Address.
	// If the Protocol is static, this property must be empty.
	// If the Protocol is bgp, the minimal parameters to be defined in bgp properties
	// are RemoteASN and LocalASN
	// +optional
	Bgp BgpSpec `json:"bgp,omitempty"`

	// Parameters to work with the static routing configured on the Gateway Router with specified Address.
	// If the Protocol is bgp, this property must be empty.
	// +optional
	Static StaticSpec `json:"static,omitempty"`
}

// BgpSpec defines the parameters to set up a BGP session
type BgpSpec struct {
	// The ASN number of the Gateway Router
	RemoteASN *uint32 `json:"remote-asn,omitempty"`

	// The ASN number of the system where the Attractor FrontEnds locates
	LocalASN *uint32 `json:"local-asn,omitempty"`

	// BFD monitoring of BGP session.
	// +optional
	BFD BfdSpec `json:"bfd,omitempty"`

	// Hold timer of the BGP session. Please refere to BGP material to understand what this implies.
	// The value must be a valid duration format. For example, 90s, 1m, 1h.
	// The duration will be rounded by second
	// Minimum duration is 3s.
	// +optional
	HoldTime string `json:"hold-time,omitempty"`

	// BGP listening port of the Gateway Router.
	// +optional
	RemotePort *uint16 `json:"remote-port,omitempty"`

	// BGP listening port of the Attractor FrontEnds.
	// +optional
	LocalPort *uint16 `json:"local-port,omitempty"`

	// BGP authentication (RFC2385).
	// +optional
	Auth *BgpAuth `json:"auth,omitempty"`
}

// StaticSpec defines the parameters to set up static routes
type StaticSpec struct {
	// BFD monitoring of Static session.
	// +optional
	BFD BfdSpec `json:"bfd,omitempty"`
}

// Bfd defines the parameters to configure the BFD session
// The static gateways shares the same interface shall define the same bfd configuration
type BfdSpec struct {
	// BFD monitoring.
	// Valid values are:
	// - false: no BFD monitoring;
	// - true: turns on the BFD monitoring.
	// When left empty, there is no BFD monitoring.
	// +optional
	Switch *bool `json:"switch,omitempty"`

	// Min-tx timer of bfd session. Please refere to BFD material to understand what this implies.
	// The value must be a valid duration format. For example, 300ms, 90s, 1m, 1h.
	// The duration will be rounded by millisecond.
	// +optional
	MinTx string `json:"min-tx,omitempty"`

	// Min-rx timer of bfd session. Please refere to BFD material to understand what this implies.
	// The value must be a valid duration format. For example, 300ms, 90s, 1m, 1h.
	// The duration will be rounded by millisecond.
	// +optional
	MinRx string `json:"min-rx,omitempty"`

	// Multiplier of bfd session.
	// When this number of bfd packets failed to receive, bfd session will go down.
	// +optional
	Multiplier *uint16 `json:"multiplier,omitempty"`
}

// BgpAuth defines the parameters to configure BGP authentication
type BgpAuth struct {
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9\.-_]+`

	// Name of the BGP authentication key, used internally as a reference.
	// KeyName is a key in the data section of a Secret. The associated value in
	// the Secret is the password (pre-shared key) to be used for authentication.
	// Must consist of alphanumeric characters, ".", "-" or "_".
	KeyName string `json:"key-name,omitempty"`

	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9](?:[A-Za-z0-9-\.]{0,61}?[A-Za-z0-9])?$`

	// Name of the kubernetes Secret containing the password (pre-shared key)
	// that can be looked up based on KeyName.
	// Must be a valid  DNS subdomain name.
	KeySource string `json:"key-source,omitempty"`
}

// GatewayStatus defines the observed state of Gateway
type GatewayStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="address",type=string,JSONPath=`.spec.address`
//+kubebuilder:printcolumn:name="protocol",type=string,JSONPath=`.spec.protocol`
//+kubebuilder:printcolumn:name="trench",type=string,JSONPath=`.metadata.labels.trench`

// Gateway is the Schema for the gateways API. It defines protocol settings
// enabling communication with Gateway Router through which traffic is to be conveyed.
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
