//
//Copyright (c) 2024 Nordix Foundation
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

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.19.1
// source: api/loadbalancer/v1/stream.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	StreamAvailabilityService_Watch_FullMethodName = "/loadbalancer.v1.StreamAvailabilityService/Watch"
)

// StreamAvailabilityServiceClient is the client API for StreamAvailabilityService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StreamAvailabilityServiceClient interface {
	Watch(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (StreamAvailabilityService_WatchClient, error)
}

type streamAvailabilityServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewStreamAvailabilityServiceClient(cc grpc.ClientConnInterface) StreamAvailabilityServiceClient {
	return &streamAvailabilityServiceClient{cc}
}

func (c *streamAvailabilityServiceClient) Watch(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (StreamAvailabilityService_WatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &StreamAvailabilityService_ServiceDesc.Streams[0], StreamAvailabilityService_Watch_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &streamAvailabilityServiceWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type StreamAvailabilityService_WatchClient interface {
	Recv() (*Response, error)
	grpc.ClientStream
}

type streamAvailabilityServiceWatchClient struct {
	grpc.ClientStream
}

func (x *streamAvailabilityServiceWatchClient) Recv() (*Response, error) {
	m := new(Response)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// StreamAvailabilityServiceServer is the server API for StreamAvailabilityService service.
// All implementations must embed UnimplementedStreamAvailabilityServiceServer
// for forward compatibility
type StreamAvailabilityServiceServer interface {
	Watch(*emptypb.Empty, StreamAvailabilityService_WatchServer) error
	mustEmbedUnimplementedStreamAvailabilityServiceServer()
}

// UnimplementedStreamAvailabilityServiceServer must be embedded to have forward compatible implementations.
type UnimplementedStreamAvailabilityServiceServer struct {
}

func (UnimplementedStreamAvailabilityServiceServer) Watch(*emptypb.Empty, StreamAvailabilityService_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}
func (UnimplementedStreamAvailabilityServiceServer) mustEmbedUnimplementedStreamAvailabilityServiceServer() {
}

// UnsafeStreamAvailabilityServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StreamAvailabilityServiceServer will
// result in compilation errors.
type UnsafeStreamAvailabilityServiceServer interface {
	mustEmbedUnimplementedStreamAvailabilityServiceServer()
}

func RegisterStreamAvailabilityServiceServer(s grpc.ServiceRegistrar, srv StreamAvailabilityServiceServer) {
	s.RegisterService(&StreamAvailabilityService_ServiceDesc, srv)
}

func _StreamAvailabilityService_Watch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(emptypb.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(StreamAvailabilityServiceServer).Watch(m, &streamAvailabilityServiceWatchServer{stream})
}

type StreamAvailabilityService_WatchServer interface {
	Send(*Response) error
	grpc.ServerStream
}

type streamAvailabilityServiceWatchServer struct {
	grpc.ServerStream
}

func (x *streamAvailabilityServiceWatchServer) Send(m *Response) error {
	return x.ServerStream.SendMsg(m)
}

// StreamAvailabilityService_ServiceDesc is the grpc.ServiceDesc for StreamAvailabilityService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var StreamAvailabilityService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "loadbalancer.v1.StreamAvailabilityService",
	HandlerType: (*StreamAvailabilityServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Watch",
			Handler:       _StreamAvailabilityService_Watch_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/loadbalancer/v1/stream.proto",
}
