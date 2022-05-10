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
// 	protoc-gen-go v1.26.0
// 	protoc        v3.19.4
// source: api/nsp/v1/targetregistry.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Target_Status int32

const (
	Target_ENABLED  Target_Status = 0
	Target_DISABLED Target_Status = 1
	Target_ANY      Target_Status = 2
)

// Enum value maps for Target_Status.
var (
	Target_Status_name = map[int32]string{
		0: "ENABLED",
		1: "DISABLED",
		2: "ANY",
	}
	Target_Status_value = map[string]int32{
		"ENABLED":  0,
		"DISABLED": 1,
		"ANY":      2,
	}
)

func (x Target_Status) Enum() *Target_Status {
	p := new(Target_Status)
	*p = x
	return p
}

func (x Target_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Target_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_api_nsp_v1_targetregistry_proto_enumTypes[0].Descriptor()
}

func (Target_Status) Type() protoreflect.EnumType {
	return &file_api_nsp_v1_targetregistry_proto_enumTypes[0]
}

func (x Target_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Target_Status.Descriptor instead.
func (Target_Status) EnumDescriptor() ([]byte, []int) {
	return file_api_nsp_v1_targetregistry_proto_rawDescGZIP(), []int{1, 0}
}

type Target_Type int32

const (
	Target_DEFAULT  Target_Type = 0
	Target_FRONTEND Target_Type = 1
)

// Enum value maps for Target_Type.
var (
	Target_Type_name = map[int32]string{
		0: "DEFAULT",
		1: "FRONTEND",
	}
	Target_Type_value = map[string]int32{
		"DEFAULT":  0,
		"FRONTEND": 1,
	}
)

func (x Target_Type) Enum() *Target_Type {
	p := new(Target_Type)
	*p = x
	return p
}

func (x Target_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Target_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_api_nsp_v1_targetregistry_proto_enumTypes[1].Descriptor()
}

func (Target_Type) Type() protoreflect.EnumType {
	return &file_api_nsp_v1_targetregistry_proto_enumTypes[1]
}

func (x Target_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Target_Type.Descriptor instead.
func (Target_Type) EnumDescriptor() ([]byte, []int) {
	return file_api_nsp_v1_targetregistry_proto_rawDescGZIP(), []int{1, 1}
}

type TargetResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Targets []*Target `protobuf:"bytes,1,rep,name=targets,proto3" json:"targets,omitempty"`
}

func (x *TargetResponse) Reset() {
	*x = TargetResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_targetregistry_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TargetResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TargetResponse) ProtoMessage() {}

func (x *TargetResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_targetregistry_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TargetResponse.ProtoReflect.Descriptor instead.
func (*TargetResponse) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_targetregistry_proto_rawDescGZIP(), []int{0}
}

func (x *TargetResponse) GetTargets() []*Target {
	if x != nil {
		return x.Targets
	}
	return nil
}

type Target struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ips     []string          `protobuf:"bytes,1,rep,name=ips,proto3" json:"ips,omitempty"`
	Context map[string]string `protobuf:"bytes,2,rep,name=context,proto3" json:"context,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Status  Target_Status     `protobuf:"varint,3,opt,name=status,proto3,enum=nsp.v1.Target_Status" json:"status,omitempty"`
	Type    Target_Type       `protobuf:"varint,4,opt,name=type,proto3,enum=nsp.v1.Target_Type" json:"type,omitempty"`
	Stream  *Stream           `protobuf:"bytes,5,opt,name=stream,proto3" json:"stream,omitempty"`
}

func (x *Target) Reset() {
	*x = Target{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_nsp_v1_targetregistry_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Target) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Target) ProtoMessage() {}

func (x *Target) ProtoReflect() protoreflect.Message {
	mi := &file_api_nsp_v1_targetregistry_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Target.ProtoReflect.Descriptor instead.
func (*Target) Descriptor() ([]byte, []int) {
	return file_api_nsp_v1_targetregistry_proto_rawDescGZIP(), []int{1}
}

func (x *Target) GetIps() []string {
	if x != nil {
		return x.Ips
	}
	return nil
}

func (x *Target) GetContext() map[string]string {
	if x != nil {
		return x.Context
	}
	return nil
}

func (x *Target) GetStatus() Target_Status {
	if x != nil {
		return x.Status
	}
	return Target_ENABLED
}

func (x *Target) GetType() Target_Type {
	if x != nil {
		return x.Type
	}
	return Target_DEFAULT
}

func (x *Target) GetStream() *Stream {
	if x != nil {
		return x.Stream
	}
	return nil
}

var File_api_nsp_v1_targetregistry_proto protoreflect.FileDescriptor

var file_api_nsp_v1_targetregistry_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x61, 0x70, 0x69, 0x2f, 0x6e, 0x73, 0x70, 0x2f, 0x76, 0x31, 0x2f, 0x74, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x06, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x16, 0x61, 0x70, 0x69, 0x2f, 0x6e, 0x73, 0x70, 0x2f,
	0x76, 0x31, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3a,
	0x0a, 0x0e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x28, 0x0a, 0x07, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x0e, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x52, 0x07, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x22, 0xde, 0x02, 0x0a, 0x06, 0x54,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x69, 0x70, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x03, 0x69, 0x70, 0x73, 0x12, 0x35, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65,
	0x78, 0x74, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76,
	0x31, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x12, 0x2d,
	0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x15,
	0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x2e, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x27, 0x0a,
	0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x13, 0x2e, 0x6e, 0x73,
	0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x2e, 0x54, 0x79, 0x70, 0x65,
	0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x26, 0x0a, 0x06, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e,
	0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x06, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x1a, 0x3a,
	0x0a, 0x0c, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x2c, 0x0a, 0x06, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x12, 0x0b, 0x0a, 0x07, 0x45, 0x4e, 0x41, 0x42, 0x4c, 0x45, 0x44, 0x10,
	0x00, 0x12, 0x0c, 0x0a, 0x08, 0x44, 0x49, 0x53, 0x41, 0x42, 0x4c, 0x45, 0x44, 0x10, 0x01, 0x12,
	0x07, 0x0a, 0x03, 0x41, 0x4e, 0x59, 0x10, 0x02, 0x22, 0x21, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65,
	0x12, 0x0b, 0x0a, 0x07, 0x44, 0x45, 0x46, 0x41, 0x55, 0x4c, 0x54, 0x10, 0x00, 0x12, 0x0c, 0x0a,
	0x08, 0x46, 0x52, 0x4f, 0x4e, 0x54, 0x45, 0x4e, 0x44, 0x10, 0x01, 0x32, 0xb3, 0x01, 0x0a, 0x0e,
	0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x12, 0x34,
	0x0a, 0x08, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x12, 0x0e, 0x2e, 0x6e, 0x73, 0x70,
	0x2e, 0x76, 0x31, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x22, 0x00, 0x12, 0x36, 0x0a, 0x0a, 0x55, 0x6e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x65, 0x72, 0x12, 0x0e, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x33, 0x0a, 0x05,
	0x57, 0x61, 0x74, 0x63, 0x68, 0x12, 0x0e, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x1a, 0x16, 0x2e, 0x6e, 0x73, 0x70, 0x2e, 0x76, 0x31, 0x2e, 0x54,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x30,
	0x01, 0x42, 0x26, 0x5a, 0x24, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x6e, 0x6f, 0x72, 0x64, 0x69, 0x78, 0x2f, 0x6d, 0x65, 0x72, 0x69, 0x64, 0x69, 0x6f, 0x2f, 0x61,
	0x70, 0x69, 0x2f, 0x6e, 0x73, 0x70, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_api_nsp_v1_targetregistry_proto_rawDescOnce sync.Once
	file_api_nsp_v1_targetregistry_proto_rawDescData = file_api_nsp_v1_targetregistry_proto_rawDesc
)

func file_api_nsp_v1_targetregistry_proto_rawDescGZIP() []byte {
	file_api_nsp_v1_targetregistry_proto_rawDescOnce.Do(func() {
		file_api_nsp_v1_targetregistry_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_nsp_v1_targetregistry_proto_rawDescData)
	})
	return file_api_nsp_v1_targetregistry_proto_rawDescData
}

var file_api_nsp_v1_targetregistry_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_api_nsp_v1_targetregistry_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_api_nsp_v1_targetregistry_proto_goTypes = []interface{}{
	(Target_Status)(0),     // 0: nsp.v1.Target.Status
	(Target_Type)(0),       // 1: nsp.v1.Target.Type
	(*TargetResponse)(nil), // 2: nsp.v1.TargetResponse
	(*Target)(nil),         // 3: nsp.v1.Target
	nil,                    // 4: nsp.v1.Target.ContextEntry
	(*Stream)(nil),         // 5: nsp.v1.Stream
	(*emptypb.Empty)(nil),  // 6: google.protobuf.Empty
}
var file_api_nsp_v1_targetregistry_proto_depIdxs = []int32{
	3, // 0: nsp.v1.TargetResponse.targets:type_name -> nsp.v1.Target
	4, // 1: nsp.v1.Target.context:type_name -> nsp.v1.Target.ContextEntry
	0, // 2: nsp.v1.Target.status:type_name -> nsp.v1.Target.Status
	1, // 3: nsp.v1.Target.type:type_name -> nsp.v1.Target.Type
	5, // 4: nsp.v1.Target.stream:type_name -> nsp.v1.Stream
	3, // 5: nsp.v1.TargetRegistry.Register:input_type -> nsp.v1.Target
	3, // 6: nsp.v1.TargetRegistry.Unregister:input_type -> nsp.v1.Target
	3, // 7: nsp.v1.TargetRegistry.Watch:input_type -> nsp.v1.Target
	6, // 8: nsp.v1.TargetRegistry.Register:output_type -> google.protobuf.Empty
	6, // 9: nsp.v1.TargetRegistry.Unregister:output_type -> google.protobuf.Empty
	2, // 10: nsp.v1.TargetRegistry.Watch:output_type -> nsp.v1.TargetResponse
	8, // [8:11] is the sub-list for method output_type
	5, // [5:8] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_api_nsp_v1_targetregistry_proto_init() }
func file_api_nsp_v1_targetregistry_proto_init() {
	if File_api_nsp_v1_targetregistry_proto != nil {
		return
	}
	file_api_nsp_v1_model_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_api_nsp_v1_targetregistry_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TargetResponse); i {
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
		file_api_nsp_v1_targetregistry_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Target); i {
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
			RawDescriptor: file_api_nsp_v1_targetregistry_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_nsp_v1_targetregistry_proto_goTypes,
		DependencyIndexes: file_api_nsp_v1_targetregistry_proto_depIdxs,
		EnumInfos:         file_api_nsp_v1_targetregistry_proto_enumTypes,
		MessageInfos:      file_api_nsp_v1_targetregistry_proto_msgTypes,
	}.Build()
	File_api_nsp_v1_targetregistry_proto = out.File
	file_api_nsp_v1_targetregistry_proto_rawDesc = nil
	file_api_nsp_v1_targetregistry_proto_goTypes = nil
	file_api_nsp_v1_targetregistry_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// TargetRegistryClient is the client API for TargetRegistry service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type TargetRegistryClient interface {
	Register(ctx context.Context, in *Target, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Unregister(ctx context.Context, in *Target, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Watch(ctx context.Context, in *Target, opts ...grpc.CallOption) (TargetRegistry_WatchClient, error)
}

type targetRegistryClient struct {
	cc grpc.ClientConnInterface
}

func NewTargetRegistryClient(cc grpc.ClientConnInterface) TargetRegistryClient {
	return &targetRegistryClient{cc}
}

func (c *targetRegistryClient) Register(ctx context.Context, in *Target, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/nsp.v1.TargetRegistry/Register", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *targetRegistryClient) Unregister(ctx context.Context, in *Target, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/nsp.v1.TargetRegistry/Unregister", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *targetRegistryClient) Watch(ctx context.Context, in *Target, opts ...grpc.CallOption) (TargetRegistry_WatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &_TargetRegistry_serviceDesc.Streams[0], "/nsp.v1.TargetRegistry/Watch", opts...)
	if err != nil {
		return nil, err
	}
	x := &targetRegistryWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type TargetRegistry_WatchClient interface {
	Recv() (*TargetResponse, error)
	grpc.ClientStream
}

type targetRegistryWatchClient struct {
	grpc.ClientStream
}

func (x *targetRegistryWatchClient) Recv() (*TargetResponse, error) {
	m := new(TargetResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// TargetRegistryServer is the server API for TargetRegistry service.
type TargetRegistryServer interface {
	Register(context.Context, *Target) (*emptypb.Empty, error)
	Unregister(context.Context, *Target) (*emptypb.Empty, error)
	Watch(*Target, TargetRegistry_WatchServer) error
}

// UnimplementedTargetRegistryServer can be embedded to have forward compatible implementations.
type UnimplementedTargetRegistryServer struct {
}

func (*UnimplementedTargetRegistryServer) Register(context.Context, *Target) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (*UnimplementedTargetRegistryServer) Unregister(context.Context, *Target) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Unregister not implemented")
}
func (*UnimplementedTargetRegistryServer) Watch(*Target, TargetRegistry_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}

func RegisterTargetRegistryServer(s *grpc.Server, srv TargetRegistryServer) {
	s.RegisterService(&_TargetRegistry_serviceDesc, srv)
}

func _TargetRegistry_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Target)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TargetRegistryServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nsp.v1.TargetRegistry/Register",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TargetRegistryServer).Register(ctx, req.(*Target))
	}
	return interceptor(ctx, in, info, handler)
}

func _TargetRegistry_Unregister_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Target)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TargetRegistryServer).Unregister(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nsp.v1.TargetRegistry/Unregister",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TargetRegistryServer).Unregister(ctx, req.(*Target))
	}
	return interceptor(ctx, in, info, handler)
}

func _TargetRegistry_Watch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Target)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(TargetRegistryServer).Watch(m, &targetRegistryWatchServer{stream})
}

type TargetRegistry_WatchServer interface {
	Send(*TargetResponse) error
	grpc.ServerStream
}

type targetRegistryWatchServer struct {
	grpc.ServerStream
}

func (x *targetRegistryWatchServer) Send(m *TargetResponse) error {
	return x.ServerStream.SendMsg(m)
}

var _TargetRegistry_serviceDesc = grpc.ServiceDesc{
	ServiceName: "nsp.v1.TargetRegistry",
	HandlerType: (*TargetRegistryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Register",
			Handler:    _TargetRegistry_Register_Handler,
		},
		{
			MethodName: "Unregister",
			Handler:    _TargetRegistry_Unregister_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Watch",
			Handler:       _TargetRegistry_Watch_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/nsp/v1/targetregistry.proto",
}
