//
//Copyright (c) 2021 Nordix Foundation
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.15.8
// source: api/nsp/v1/model.proto

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Trench struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the trench
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *Trench) Reset() {
	*x = Trench{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Trench) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Trench) ProtoMessage() {}

func (x *Trench) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Trench.ProtoReflect.Descriptor instead.
func (*Trench) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{0}
}

func (x *Trench) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type Conduit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the conduit
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Trench the conduit belongs to
	Trench *Trench `protobuf:"bytes,2,opt,name=trench,proto3" json:"trench,omitempty"`
}

func (x *Conduit) Reset() {
	*x = Conduit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Conduit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Conduit) ProtoMessage() {}

func (x *Conduit) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Conduit.ProtoReflect.Descriptor instead.
func (*Conduit) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{1}
}

func (x *Conduit) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Conduit) GetTrench() *Trench {
	if x != nil {
		return x.Trench
	}
	return nil
}

type Stream struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the stream
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Conduit the stream belongs to
	Conduit *Conduit `protobuf:"bytes,2,opt,name=conduit,proto3" json:"conduit,omitempty"`
}

func (x *Stream) Reset() {
	*x = Stream{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Stream) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Stream) ProtoMessage() {}

func (x *Stream) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Stream.ProtoReflect.Descriptor instead.
func (*Stream) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{2}
}

func (x *Stream) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Stream) GetConduit() *Conduit {
	if x != nil {
		return x.Conduit
	}
	return nil
}

type Flow struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the flow
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Source subnets allowed in the flow
	// e.g.: ["124.0.0.0/24", "2001::/32"]
	SourceSubnets []string `protobuf:"bytes,2,rep,name=sourceSubnets,proto3" json:"sourceSubnets,omitempty"`
	// Destination port ranges allowed in the flow
	// e.g.: ["80", "90-95"]
	DestinationPortRanges []string `protobuf:"bytes,3,rep,name=destinationPortRanges,proto3" json:"destinationPortRanges,omitempty"`
	// Source port ranges allowed in the flow
	// e.g.: ["35000-35500", "40000"]
	SourcePortRanges []string `protobuf:"bytes,4,rep,name=sourcePortRanges,proto3" json:"sourcePortRanges,omitempty"`
	// Protocols allowed
	// e.g.: ["tcp", "udp"]
	Protocols []string `protobuf:"bytes,5,rep,name=protocols,proto3" json:"protocols,omitempty"`
	// Priority of the flow
	Priority int32 `protobuf:"varint,6,opt,name=priority,proto3" json:"priority,omitempty"`
	// Stream the flow belongs to
	Stream *Stream `protobuf:"bytes,7,opt,name=stream,proto3" json:"stream,omitempty"`
	Vips   []*Vip  `protobuf:"bytes,8,rep,name=vips,proto3" json:"vips,omitempty"`
}

func (x *Flow) Reset() {
	*x = Flow{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Flow) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Flow) ProtoMessage() {}

func (x *Flow) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Flow.ProtoReflect.Descriptor instead.
func (*Flow) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{3}
}

func (x *Flow) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Flow) GetSourceSubnets() []string {
	if x != nil {
		return x.SourceSubnets
	}
	return nil
}

func (x *Flow) GetDestinationPortRanges() []string {
	if x != nil {
		return x.DestinationPortRanges
	}
	return nil
}

func (x *Flow) GetSourcePortRanges() []string {
	if x != nil {
		return x.SourcePortRanges
	}
	return nil
}

func (x *Flow) GetProtocols() []string {
	if x != nil {
		return x.Protocols
	}
	return nil
}

func (x *Flow) GetPriority() int32 {
	if x != nil {
		return x.Priority
	}
	return 0
}

func (x *Flow) GetStream() *Stream {
	if x != nil {
		return x.Stream
	}
	return nil
}

func (x *Flow) GetVips() []*Vip {
	if x != nil {
		return x.Vips
	}
	return nil
}

type Vip struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the vip
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// vip address
	// e.g.: 124.0.0.0/24 or 2001::/32
	Address string `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	// Trench the vip belongs to
	Trench *Trench `protobuf:"bytes,3,opt,name=trench,proto3" json:"trench,omitempty"`
}

func (x *Vip) Reset() {
	*x = Vip{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Vip) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Vip) ProtoMessage() {}

func (x *Vip) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Vip.ProtoReflect.Descriptor instead.
func (*Vip) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{4}
}

func (x *Vip) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Vip) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *Vip) GetTrench() *Trench {
	if x != nil {
		return x.Trench
	}
	return nil
}

type Attractor struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the attractor
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Trench the attractor belongs to
	Trench   *Trench    `protobuf:"bytes,2,opt,name=trench,proto3" json:"trench,omitempty"`
	Vips     []*Vip     `protobuf:"bytes,3,rep,name=vips,proto3" json:"vips,omitempty"`
	Gateways []*Gateway `protobuf:"bytes,4,rep,name=gateways,proto3" json:"gateways,omitempty"`
}

func (x *Attractor) Reset() {
	*x = Attractor{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Attractor) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Attractor) ProtoMessage() {}

func (x *Attractor) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Attractor.ProtoReflect.Descriptor instead.
func (*Attractor) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{5}
}

func (x *Attractor) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Attractor) GetTrench() *Trench {
	if x != nil {
		return x.Trench
	}
	return nil
}

func (x *Attractor) GetVips() []*Vip {
	if x != nil {
		return x.Vips
	}
	return nil
}

func (x *Attractor) GetGateways() []*Gateway {
	if x != nil {
		return x.Gateways
	}
	return nil
}

type Gateway struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the vip
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// address of the gateway
	// e.g.: 124.0.0.0/24 or 2001::/32
	Address    string `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	RemoteASN  uint32 `protobuf:"varint,3,opt,name=remoteASN,proto3" json:"remoteASN,omitempty"`
	LocalASN   uint32 `protobuf:"varint,4,opt,name=localASN,proto3" json:"localASN,omitempty"`
	RemotePort uint32 `protobuf:"varint,5,opt,name=remotePort,proto3" json:"remotePort,omitempty"`
	LocalPort  uint32 `protobuf:"varint,6,opt,name=localPort,proto3" json:"localPort,omitempty"`
	IpFamily   string `protobuf:"bytes,7,opt,name=ipFamily,proto3" json:"ipFamily,omitempty"`
	Bfd        bool   `protobuf:"varint,8,opt,name=bfd,proto3" json:"bfd,omitempty"`
	Protocol   string `protobuf:"bytes,9,opt,name=protocol,proto3" json:"protocol,omitempty"`
	HoldTime   uint32 `protobuf:"varint,10,opt,name=holdTime,proto3" json:"holdTime,omitempty"`
	// Trench the gateway belongs to
	Trench *Trench `protobuf:"bytes,11,opt,name=trench,proto3" json:"trench,omitempty"`
}

func (x *Gateway) Reset() {
	*x = Gateway{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Gateway) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Gateway) ProtoMessage() {}

func (x *Gateway) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Gateway.ProtoReflect.Descriptor instead.
func (*Gateway) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{6}
}

func (x *Gateway) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Gateway) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *Gateway) GetRemoteASN() uint32 {
	if x != nil {
		return x.RemoteASN
	}
	return 0
}

func (x *Gateway) GetLocalASN() uint32 {
	if x != nil {
		return x.LocalASN
	}
	return 0
}

func (x *Gateway) GetRemotePort() uint32 {
	if x != nil {
		return x.RemotePort
	}
	return 0
}

func (x *Gateway) GetLocalPort() uint32 {
	if x != nil {
		return x.LocalPort
	}
	return 0
}

func (x *Gateway) GetIpFamily() string {
	if x != nil {
		return x.IpFamily
	}
	return ""
}

func (x *Gateway) GetBfd() bool {
	if x != nil {
		return x.Bfd
	}
	return false
}

func (x *Gateway) GetProtocol() string {
	if x != nil {
		return x.Protocol
	}
	return ""
}

func (x *Gateway) GetHoldTime() uint32 {
	if x != nil {
		return x.HoldTime
	}
	return 0
}

func (x *Gateway) GetTrench() *Trench {
	if x != nil {
		return x.Trench
	}
	return nil
}

type TrenchResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Trench *Trench `protobuf:"bytes,1,opt,name=trench,proto3" json:"trench,omitempty"`
}

func (x *TrenchResponse) Reset() {
	*x = TrenchResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TrenchResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TrenchResponse) ProtoMessage() {}

func (x *TrenchResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TrenchResponse.ProtoReflect.Descriptor instead.
func (*TrenchResponse) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{7}
}

func (x *TrenchResponse) GetTrench() *Trench {
	if x != nil {
		return x.Trench
	}
	return nil
}

type ConduitResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Conduits []*Conduit `protobuf:"bytes,1,rep,name=conduits,proto3" json:"conduits,omitempty"`
}

func (x *ConduitResponse) Reset() {
	*x = ConduitResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConduitResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConduitResponse) ProtoMessage() {}

func (x *ConduitResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConduitResponse.ProtoReflect.Descriptor instead.
func (*ConduitResponse) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{8}
}

func (x *ConduitResponse) GetConduits() []*Conduit {
	if x != nil {
		return x.Conduits
	}
	return nil
}

type StreamResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Streams []*Stream `protobuf:"bytes,1,rep,name=streams,proto3" json:"streams,omitempty"`
}

func (x *StreamResponse) Reset() {
	*x = StreamResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StreamResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StreamResponse) ProtoMessage() {}

func (x *StreamResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StreamResponse.ProtoReflect.Descriptor instead.
func (*StreamResponse) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{9}
}

func (x *StreamResponse) GetStreams() []*Stream {
	if x != nil {
		return x.Streams
	}
	return nil
}

type FlowResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Flows []*Flow `protobuf:"bytes,1,rep,name=flows,proto3" json:"flows,omitempty"`
}

func (x *FlowResponse) Reset() {
	*x = FlowResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FlowResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FlowResponse) ProtoMessage() {}

func (x *FlowResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FlowResponse.ProtoReflect.Descriptor instead.
func (*FlowResponse) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{10}
}

func (x *FlowResponse) GetFlows() []*Flow {
	if x != nil {
		return x.Flows
	}
	return nil
}

type VipResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Vips []*Vip `protobuf:"bytes,1,rep,name=vips,proto3" json:"vips,omitempty"`
}

func (x *VipResponse) Reset() {
	*x = VipResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[11]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VipResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VipResponse) ProtoMessage() {}

func (x *VipResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[11]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VipResponse.ProtoReflect.Descriptor instead.
func (*VipResponse) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{11}
}

func (x *VipResponse) GetVips() []*Vip {
	if x != nil {
		return x.Vips
	}
	return nil
}

type AttractorResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Attractors []*Attractor `protobuf:"bytes,1,rep,name=attractors,proto3" json:"attractors,omitempty"`
}

func (x *AttractorResponse) Reset() {
	*x = AttractorResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[12]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttractorResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttractorResponse) ProtoMessage() {}

func (x *AttractorResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[12]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttractorResponse.ProtoReflect.Descriptor instead.
func (*AttractorResponse) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{12}
}

func (x *AttractorResponse) GetAttractors() []*Attractor {
	if x != nil {
		return x.Attractors
	}
	return nil
}

type GatewayResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Gateways []*Gateway `protobuf:"bytes,1,rep,name=gateways,proto3" json:"gateways,omitempty"`
}

func (x *GatewayResponse) Reset() {
	*x = GatewayResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_model_proto_msgTypes[13]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GatewayResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GatewayResponse) ProtoMessage() {}

func (x *GatewayResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_model_proto_msgTypes[13]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GatewayResponse.ProtoReflect.Descriptor instead.
func (*GatewayResponse) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_model_proto_rawDescGZIP(), []int{13}
}

func (x *GatewayResponse) GetGateways() []*Gateway {
	if x != nil {
		return x.Gateways
	}
	return nil
}

var File_api_nsp_v1_model_proto protoreflect.FileDescriptor

var file_api_nsp_v1_model_proto_rawDesc = []byte{
	0x0a, 0x16, 0x61, 0x70, 0x69, 0x2f, 0x6e, 0x73, 0x70, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x6f, 0x64,
	0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31,
	0x22, 0x1c, 0x0a, 0x06, 0x54, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x45,
	0x0a, 0x07, 0x43, 0x6f, 0x6e, 0x64, 0x75, 0x69, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x26, 0x0a,
	0x06, 0x74, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e,
	0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x52, 0x06, 0x74,
	0x72, 0x65, 0x6e, 0x63, 0x68, 0x22, 0x47, 0x0a, 0x06, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x29, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x64, 0x75, 0x69, 0x74, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f,
	0x6e, 0x64, 0x75, 0x69, 0x74, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x64, 0x75, 0x69, 0x74, 0x22, 0xa5,
	0x02, 0x0a, 0x04, 0x46, 0x6c, 0x6f, 0x77, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x24, 0x0a, 0x0d, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x53, 0x75, 0x62, 0x6e, 0x65, 0x74, 0x73, 0x18, 0x02, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x0d, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x53, 0x75, 0x62, 0x6e, 0x65, 0x74,
	0x73, 0x12, 0x34, 0x0a, 0x15, 0x64, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x50, 0x6f, 0x72, 0x74, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x15, 0x64, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x50, 0x6f, 0x72,
	0x74, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x73, 0x12, 0x2a, 0x0a, 0x10, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x50, 0x6f, 0x72, 0x74, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x10, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50, 0x6f, 0x72, 0x74, 0x52, 0x61, 0x6e,
	0x67, 0x65, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x73,
	0x18, 0x05, 0x20, 0x03, 0x28, 0x09, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c,
	0x73, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x72, 0x69, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x08, 0x70, 0x72, 0x69, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x12, 0x26, 0x0a,
	0x06, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e,
	0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x06, 0x73,
	0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x1f, 0x0a, 0x04, 0x76, 0x69, 0x70, 0x73, 0x18, 0x08, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x56, 0x69, 0x70,
	0x52, 0x04, 0x76, 0x69, 0x70, 0x73, 0x22, 0x5b, 0x0a, 0x03, 0x56, 0x69, 0x70, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x26, 0x0a, 0x06, 0x74,
	0x72, 0x65, 0x6e, 0x63, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x6e, 0x73,
	0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x52, 0x06, 0x74, 0x72, 0x65,
	0x6e, 0x63, 0x68, 0x22, 0x95, 0x01, 0x0a, 0x09, 0x41, 0x74, 0x74, 0x72, 0x61, 0x63, 0x74, 0x6f,
	0x72, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x26, 0x0a, 0x06, 0x74, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54,
	0x72, 0x65, 0x6e, 0x63, 0x68, 0x52, 0x06, 0x74, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x12, 0x1f, 0x0a,
	0x04, 0x76, 0x69, 0x70, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x6e, 0x73,
	0x70, 0x2e, 0x76, 0x31, 0x2e, 0x56, 0x69, 0x70, 0x52, 0x04, 0x76, 0x69, 0x70, 0x73, 0x12, 0x2b,
	0x0a, 0x08, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x0f, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61,
	0x79, 0x52, 0x08, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x73, 0x22, 0xbd, 0x02, 0x0a, 0x07,
	0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x61,
	0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64,
	0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x41,
	0x53, 0x4e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65,
	0x41, 0x53, 0x4e, 0x12, 0x1a, 0x0a, 0x08, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x41, 0x53, 0x4e, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x41, 0x53, 0x4e, 0x12,
	0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x50, 0x6f, 0x72, 0x74, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x0a, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x50, 0x6f, 0x72, 0x74, 0x12,
	0x1c, 0x0a, 0x09, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x50, 0x6f, 0x72, 0x74, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x09, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x50, 0x6f, 0x72, 0x74, 0x12, 0x1a, 0x0a,
	0x08, 0x69, 0x70, 0x46, 0x61, 0x6d, 0x69, 0x6c, 0x79, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x69, 0x70, 0x46, 0x61, 0x6d, 0x69, 0x6c, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x62, 0x66, 0x64,
	0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x52, 0x03, 0x62, 0x66, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x12, 0x1a, 0x0a, 0x08, 0x68, 0x6f, 0x6c, 0x64, 0x54,
	0x69, 0x6d, 0x65, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x68, 0x6f, 0x6c, 0x64, 0x54,
	0x69, 0x6d, 0x65, 0x12, 0x26, 0x0a, 0x06, 0x74, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x18, 0x0b, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x72, 0x65,
	0x6e, 0x63, 0x68, 0x52, 0x06, 0x74, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x22, 0x38, 0x0a, 0x0e, 0x54,
	0x72, 0x65, 0x6e, 0x63, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x26, 0x0a,
	0x06, 0x74, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e,
	0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x72, 0x65, 0x6e, 0x63, 0x68, 0x52, 0x06, 0x74,
	0x72, 0x65, 0x6e, 0x63, 0x68, 0x22, 0x3e, 0x0a, 0x0f, 0x43, 0x6f, 0x6e, 0x64, 0x75, 0x69, 0x74,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x2b, 0x0a, 0x08, 0x63, 0x6f, 0x6e, 0x64,
	0x75, 0x69, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x6e, 0x73, 0x70,
	0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6e, 0x64, 0x75, 0x69, 0x74, 0x52, 0x08, 0x63, 0x6f, 0x6e,
	0x64, 0x75, 0x69, 0x74, 0x73, 0x22, 0x3a, 0x0a, 0x0e, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x28, 0x0a, 0x07, 0x73, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76,
	0x31, 0x2e, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x07, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x73, 0x22, 0x32, 0x0a, 0x0c, 0x46, 0x6c, 0x6f, 0x77, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x22, 0x0a, 0x05, 0x66, 0x6c, 0x6f, 0x77, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x0c, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x52, 0x05,
	0x66, 0x6c, 0x6f, 0x77, 0x73, 0x22, 0x2e, 0x0a, 0x0b, 0x56, 0x69, 0x70, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1f, 0x0a, 0x04, 0x76, 0x69, 0x70, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x56, 0x69, 0x70, 0x52,
	0x04, 0x76, 0x69, 0x70, 0x73, 0x22, 0x46, 0x0a, 0x11, 0x41, 0x74, 0x74, 0x72, 0x61, 0x63, 0x74,
	0x6f, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x31, 0x0a, 0x0a, 0x61, 0x74,
	0x74, 0x72, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11,
	0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x74, 0x74, 0x72, 0x61, 0x63, 0x74, 0x6f,
	0x72, 0x52, 0x0a, 0x61, 0x74, 0x74, 0x72, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x22, 0x3e, 0x0a,
	0x0f, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x2b, 0x0a, 0x08, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x61, 0x74, 0x65,
	0x77, 0x61, 0x79, 0x52, 0x08, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x73, 0x42, 0x26, 0x5a,
	0x24, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6e, 0x6f, 0x72, 0x64,
	0x69, 0x78, 0x2f, 0x6d, 0x65, 0x72, 0x69, 0x64, 0x69, 0x6f, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6e,
	0x73, 0x70, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_nsp_v1_model_proto_rawDescOnce sync.Once
	file_api_nsp_v1_model_proto_rawDescData = file_api_nsp_v1_model_proto_rawDesc
)

func file_api_nsp_v1_model_proto_rawDescGZIP() []byte {
	file_api_nsp_v1_model_proto_rawDescOnce.Do(func() {
		file_api_nsp_v1_model_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_nsp_v1_model_proto_rawDescData)
	})
	return file_api_nsp_v1_model_proto_rawDescData
}

var file_api_nsp_v1_model_proto_msgTypes = make([]protoimpl.MessageInfo, 14)
var file_api_nsp_v1_model_proto_goTypes = []interface{}{
	(*Trench)(nil),            // 0: nsp.v1.Trench
	(*Conduit)(nil),           // 1: nsp.v1.Conduit
	(*Stream)(nil),            // 2: nsp.v1.Stream
	(*Flow)(nil),              // 3: nsp.v1.Flow
	(*Vip)(nil),               // 4: nsp.v1.Vip
	(*Attractor)(nil),         // 5: nsp.v1.Attractor
	(*Gateway)(nil),           // 6: nsp.v1.Gateway
	(*TrenchResponse)(nil),    // 7: nsp.v1.TrenchResponse
	(*ConduitResponse)(nil),   // 8: nsp.v1.ConduitResponse
	(*StreamResponse)(nil),    // 9: nsp.v1.StreamResponse
	(*FlowResponse)(nil),      // 10: nsp.v1.FlowResponse
	(*VipResponse)(nil),       // 11: nsp.v1.VipResponse
	(*AttractorResponse)(nil), // 12: nsp.v1.AttractorResponse
	(*GatewayResponse)(nil),   // 13: nsp.v1.GatewayResponse
}
var file_api_nsp_v1_model_proto_depIdxs = []int32{
	0,  // 0: nsp.v1.Conduit.trench:type_name -> nsp.v1.Trench
	1,  // 1: nsp.v1.Stream.conduit:type_name -> nsp.v1.Conduit
	2,  // 2: nsp.v1.Flow.stream:type_name -> nsp.v1.Stream
	4,  // 3: nsp.v1.Flow.vips:type_name -> nsp.v1.Vip
	0,  // 4: nsp.v1.Vip.trench:type_name -> nsp.v1.Trench
	0,  // 5: nsp.v1.Attractor.trench:type_name -> nsp.v1.Trench
	4,  // 6: nsp.v1.Attractor.vips:type_name -> nsp.v1.Vip
	6,  // 7: nsp.v1.Attractor.gateways:type_name -> nsp.v1.Gateway
	0,  // 8: nsp.v1.Gateway.trench:type_name -> nsp.v1.Trench
	0,  // 9: nsp.v1.TrenchResponse.trench:type_name -> nsp.v1.Trench
	1,  // 10: nsp.v1.ConduitResponse.conduits:type_name -> nsp.v1.Conduit
	2,  // 11: nsp.v1.StreamResponse.streams:type_name -> nsp.v1.Stream
	3,  // 12: nsp.v1.FlowResponse.flows:type_name -> nsp.v1.Flow
	4,  // 13: nsp.v1.VipResponse.vips:type_name -> nsp.v1.Vip
	5,  // 14: nsp.v1.AttractorResponse.attractors:type_name -> nsp.v1.Attractor
	6,  // 15: nsp.v1.GatewayResponse.gateways:type_name -> nsp.v1.Gateway
	16, // [16:16] is the sub-list for method output_type
	16, // [16:16] is the sub-list for method input_type
	16, // [16:16] is the sub-list for extension type_name
	16, // [16:16] is the sub-list for extension extendee
	0,  // [0:16] is the sub-list for field type_name
}

func init() { file_api_nsp_v1_model_proto_init() }
func file_api_nsp_v1_model_proto_init() {
	if File_api_nsp_v1_model_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_nsp_v1_model_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Trench); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Conduit); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Stream); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Flow); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Vip); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Attractor); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Gateway); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TrenchResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConduitResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StreamResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FlowResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[11].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VipResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[12].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttractorResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_nsp_v1_model_proto_msgTypes[13].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GatewayResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_nsp_v1_model_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   14,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_nsp_v1_model_proto_goTypes,
		DependencyIndexes: file_api_nsp_v1_model_proto_depIdxs,
		MessageInfos:      file_api_nsp_v1_model_proto_msgTypes,
	}.Build()
	File_api_nsp_v1_model_proto = out.File
	file_api_nsp_v1_model_proto_rawDesc = nil
	file_api_nsp_v1_model_proto_goTypes = nil
	file_api_nsp_v1_model_proto_depIdxs = nil
}
