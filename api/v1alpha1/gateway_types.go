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

// GatewaySpec defines the desired state of Gateway
type GatewaySpec struct {
	Address  string     `json:"address"`
	Protocol string     `json:"protocol,omitempty"`
	Bgp      BgpSpec    `json:"bgp,omitempty"`
	Static   StaticSpec `json:"static,omitempty"`
}

type BgpSpec struct {
	// (mandatory) The ASN number of the Gateway
	RemoteASN *uint32 `json:"remote-asn,omitempty"`
	// (mandatory) The ASN number of the system where the FrontEnd locates
	LocalASN *uint32 `json:"local-asn,omitempty"`
	// (optional) BFD monitoring of BGP session. Default "false"
	BFD *bool `json:"bfd,omitempty"`
	// (optional) Hold timer of the BGP session. Default 240s
	HoldTime string `json:"hold-time,omitempty"`
	// (optional) BGP listening port of the gateway. Default 179
	RemotePort *uint16 `json:"remote-port,omitempty"`
	// (optional) BGP listening port of the FrontEnd. Default 179
	LocalPort *uint16 `json:"local-port,omitempty"`
}

type StaticSpec struct {
}

// GatewayStatus defines the observed state of Gateway
type GatewayStatus struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="address",type=string,JSONPath=`.spec.address`
//+kubebuilder:printcolumn:name="protocol",type=string,JSONPath=`.spec.protocol`
//+kubebuilder:printcolumn:name="attractor",type=string,JSONPath=`.metadata.labels.attractor`
//+kubebuilder:printcolumn:name="status",type=string,JSONPath=`.status.status`
//+kubebuilder:printcolumn:name="message",type=string,JSONPath=`.status.message`
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
