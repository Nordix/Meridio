/*
Copyright (c) 2022 Nordix Foundation

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// AttractorSpec defines the desired state of Attractor
type AttractorSpec struct {
	// +kubebuilder:default=1

	// The number of front-end pods. (The load-balancer is bundled with front-end currently)
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Reference to the composite conduits
	Composites []string `json:"composites"`

	// gateways that attractor expect to use
	// +optional
	Gateways []string `json:"gateways,omitempty"`

	// vips that attractor will announce to the gateways when possible
	// +optional
	Vips []string `json:"vips,omitempty"`

	// defines the interface information that attractor use
	Interface InterfaceSpec `json:"interface"`
}

type InterfaceSpec struct {
	// Name of the interface.
	// Must be a valid Linux kernel interface name.
	// +kubebuilder:validation:Pattern=`^[^:\//\s]{1,13}$`
	Name string `json:"name"`

	// (immutable) IPv4 prefix of the interface, which is used for frontend to set up communication with the ipv4 gateways.
	// If the type is "nsm-vlan", this information must be specified.
	PrefixIPv4 string `json:"ipv4-prefix,omitempty"`

	// (immutable) IPv6 prefix of the interface, which is used for frontend to set up communication with the ipv6 gateways.
	// If the type is "nsm-vlan", this information must be specified.
	PrefixIPv6 string `json:"ipv6-prefix,omitempty"`

	// Interface choice.
	// +kubebuilder:default=nsm-vlan
	// +kubebuilder:validation:Enum=nsm-vlan;network-attachment
	Type string `json:"type,omitempty"`

	// If the type is "nsm-vlan", this information must be specified.
	NSMVlan NSMVlanSpec `json:"nsm-vlan,omitempty"`

	// If the type is "network-attachment", this information must be specified.
	// One NetworkAttachmentSpec allowed currently.
	NetworkAttachments []*NetworkAttachmentSpec `json:"network-attachments,omitempty"`
}

type NSMVlanSpec struct {
	// (immutable) master interface of the vlan interface to be used for external connectivity
	BaseInterface string `json:"base-interface,omitempty"`

	// (immutable) vlan ID of the vlan interface to be used for external connectivity
	VlanID *int32 `json:"vlan-id,omitempty"`
}

// NetworkAttachmentSpec identifies a Network Attachment Definition intended to setup an interface.
// It is a subset of NetworkSelectionElement:
// https://pkg.go.dev/github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1#NetworkSelectionElement
type NetworkAttachmentSpec struct {
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength=253

	// Name of the Network Attachment Definition.
	// Must be a valid lowercase RFC 1123 subdomain.
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[a-z0-9](?:[a-z0-9-]{0,61}?[a-z0-9])?$`
	// +kubebuilder:default=default

	// Kubernetes namespace where the Network Attachment Definition is defined.
	// Must be a valid lowercase RFC 1123 label. Its default value is "default".
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^[^:\//\s]{1,13}$`

	// Name to be used as the given name for the POD interface.
	// Must be a valid Linux kernel interface name.
	// +optional
	InterfaceRequest string `json:"interface,omitempty"`
}

// AttractorStatus defines the observed state of Attractor
type AttractorStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Interface-Name",type=string,JSONPath=`.spec.interface.name`
//+kubebuilder:printcolumn:name="Interface-Type",type=string,JSONPath=`.spec.interface.type`
//+kubebuilder:printcolumn:name="Gateways",type=string,JSONPath=`.spec.gateways`
//+kubebuilder:printcolumn:name="Vips",type=string,JSONPath=`.spec.vips`
//+kubebuilder:printcolumn:name="Composites",type=string,JSONPath=`.spec.composites`
//+kubebuilder:printcolumn:name="Replicas",type=string,JSONPath=`.spec.replicas`
//+kubebuilder:printcolumn:name="Trench",type=string,JSONPath=`.metadata.labels.trench`

// Attractor is the Schema for the attractors API. It defines how traffic are
// attracted and lead into the K8s cluster. This includes which external interface
// to consume. The Attractor is instantiated as a set of pods running frontend
// functionality.
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

func init() {
	SchemeBuilder.Register(&Attractor{}, &AttractorList{})
}

func (r *Attractor) GroupResource() schema.GroupResource {
	return schema.GroupResource{
		Group:    r.GroupVersionKind().Group,
		Resource: r.GroupVersionKind().Kind,
	}
}

func (r *Attractor) GroupKind() schema.GroupKind {
	return schema.GroupKind{
		Group: r.GroupVersionKind().Group,
		Kind:  r.GroupVersionKind().Kind,
	}
}
