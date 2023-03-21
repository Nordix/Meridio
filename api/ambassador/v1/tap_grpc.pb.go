//
//Copyright (c) 2021-2022 Nordix Foundation
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
// - protoc             v3.15.8
// source: api/ambassador/v1/tap.proto

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
	Tap_Open_FullMethodName  = "/ambassador.v1.Tap/Open"
	Tap_Close_FullMethodName = "/ambassador.v1.Tap/Close"
	Tap_Watch_FullMethodName = "/ambassador.v1.Tap/Watch"
)

// TapClient is the client API for Tap service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TapClient interface {
	// Open a stream registers the target to the NSP,
	// If the trench or conduit is not connected to the target, then it will
	// be connected automatically before registering the target to the NSP.
	// If any property is not defined (empty name, nil trench/conduit...),
	// or, if another trench is already connected, an error will be returned.
	Open(ctx context.Context, in *Stream, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// Close a stream unregisters the target from the NSP, disconnects
	// the target from the conduit if no more stream is connected to it,
	// and disconnects from the trench if no more conduit is connected to it.
	// If any property is not defined (empty name, nil trench/conduit...),
	// an error will be returned.
	Close(ctx context.Context, in *Stream, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// WatchStream will return a list of stream status containing
	// the same properties as the one in parameter (nil properties
	// will be ignored). On any event (any stream created/deleted/updated)
	// the list will be sent again.
	Watch(ctx context.Context, in *Stream, opts ...grpc.CallOption) (Tap_WatchClient, error)
}

type tapClient struct {
	cc grpc.ClientConnInterface
}

func NewTapClient(cc grpc.ClientConnInterface) TapClient {
	return &tapClient{cc}
}

func (c *tapClient) Open(ctx context.Context, in *Stream, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Tap_Open_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tapClient) Close(ctx context.Context, in *Stream, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Tap_Close_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tapClient) Watch(ctx context.Context, in *Stream, opts ...grpc.CallOption) (Tap_WatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &Tap_ServiceDesc.Streams[0], Tap_Watch_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &tapWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Tap_WatchClient interface {
	Recv() (*StreamResponse, error)
	grpc.ClientStream
}

type tapWatchClient struct {
	grpc.ClientStream
}

func (x *tapWatchClient) Recv() (*StreamResponse, error) {
	m := new(StreamResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// TapServer is the server API for Tap service.
// All implementations must embed UnimplementedTapServer
// for forward compatibility
type TapServer interface {
	// Open a stream registers the target to the NSP,
	// If the trench or conduit is not connected to the target, then it will
	// be connected automatically before registering the target to the NSP.
	// If any property is not defined (empty name, nil trench/conduit...),
	// or, if another trench is already connected, an error will be returned.
	Open(context.Context, *Stream) (*emptypb.Empty, error)
	// Close a stream unregisters the target from the NSP, disconnects
	// the target from the conduit if no more stream is connected to it,
	// and disconnects from the trench if no more conduit is connected to it.
	// If any property is not defined (empty name, nil trench/conduit...),
	// an error will be returned.
	Close(context.Context, *Stream) (*emptypb.Empty, error)
	// WatchStream will return a list of stream status containing
	// the same properties as the one in parameter (nil properties
	// will be ignored). On any event (any stream created/deleted/updated)
	// the list will be sent again.
	Watch(*Stream, Tap_WatchServer) error
	mustEmbedUnimplementedTapServer()
}

// UnimplementedTapServer must be embedded to have forward compatible implementations.
type UnimplementedTapServer struct {
}

func (UnimplementedTapServer) Open(context.Context, *Stream) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Open not implemented")
}
func (UnimplementedTapServer) Close(context.Context, *Stream) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Close not implemented")
}
func (UnimplementedTapServer) Watch(*Stream, Tap_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}
func (UnimplementedTapServer) mustEmbedUnimplementedTapServer() {}

// UnsafeTapServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TapServer will
// result in compilation errors.
type UnsafeTapServer interface {
	mustEmbedUnimplementedTapServer()
}

func RegisterTapServer(s grpc.ServiceRegistrar, srv TapServer) {
	s.RegisterService(&Tap_ServiceDesc, srv)
}

func _Tap_Open_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Stream)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TapServer).Open(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Tap_Open_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TapServer).Open(ctx, req.(*Stream))
	}
	return interceptor(ctx, in, info, handler)
}

func _Tap_Close_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Stream)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TapServer).Close(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Tap_Close_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TapServer).Close(ctx, req.(*Stream))
	}
	return interceptor(ctx, in, info, handler)
}

func _Tap_Watch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Stream)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(TapServer).Watch(m, &tapWatchServer{stream})
}

type Tap_WatchServer interface {
	Send(*StreamResponse) error
	grpc.ServerStream
}

type tapWatchServer struct {
	grpc.ServerStream
}

func (x *tapWatchServer) Send(m *StreamResponse) error {
	return x.ServerStream.SendMsg(m)
}

// Tap_ServiceDesc is the grpc.ServiceDesc for Tap service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Tap_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ambassador.v1.Tap",
	HandlerType: (*TapServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Open",
			Handler:    _Tap_Open_Handler,
		},
		{
			MethodName: "Close",
			Handler:    _Tap_Close_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Watch",
			Handler:       _Tap_Watch_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/ambassador/v1/tap.proto",
}
