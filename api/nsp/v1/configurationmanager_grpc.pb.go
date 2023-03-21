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
// source: api/nsp/v1/configurationmanager.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	ConfigurationManager_WatchTrench_FullMethodName    = "/nsp.v1.ConfigurationManager/WatchTrench"
	ConfigurationManager_WatchConduit_FullMethodName   = "/nsp.v1.ConfigurationManager/WatchConduit"
	ConfigurationManager_WatchStream_FullMethodName    = "/nsp.v1.ConfigurationManager/WatchStream"
	ConfigurationManager_WatchFlow_FullMethodName      = "/nsp.v1.ConfigurationManager/WatchFlow"
	ConfigurationManager_WatchVip_FullMethodName       = "/nsp.v1.ConfigurationManager/WatchVip"
	ConfigurationManager_WatchAttractor_FullMethodName = "/nsp.v1.ConfigurationManager/WatchAttractor"
	ConfigurationManager_WatchGateway_FullMethodName   = "/nsp.v1.ConfigurationManager/WatchGateway"
)

// ConfigurationManagerClient is the client API for ConfigurationManager service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ConfigurationManagerClient interface {
	WatchTrench(ctx context.Context, in *Trench, opts ...grpc.CallOption) (ConfigurationManager_WatchTrenchClient, error)
	WatchConduit(ctx context.Context, in *Conduit, opts ...grpc.CallOption) (ConfigurationManager_WatchConduitClient, error)
	WatchStream(ctx context.Context, in *Stream, opts ...grpc.CallOption) (ConfigurationManager_WatchStreamClient, error)
	WatchFlow(ctx context.Context, in *Flow, opts ...grpc.CallOption) (ConfigurationManager_WatchFlowClient, error)
	WatchVip(ctx context.Context, in *Vip, opts ...grpc.CallOption) (ConfigurationManager_WatchVipClient, error)
	WatchAttractor(ctx context.Context, in *Attractor, opts ...grpc.CallOption) (ConfigurationManager_WatchAttractorClient, error)
	WatchGateway(ctx context.Context, in *Gateway, opts ...grpc.CallOption) (ConfigurationManager_WatchGatewayClient, error)
}

type configurationManagerClient struct {
	cc grpc.ClientConnInterface
}

func NewConfigurationManagerClient(cc grpc.ClientConnInterface) ConfigurationManagerClient {
	return &configurationManagerClient{cc}
}

func (c *configurationManagerClient) WatchTrench(ctx context.Context, in *Trench, opts ...grpc.CallOption) (ConfigurationManager_WatchTrenchClient, error) {
	stream, err := c.cc.NewStream(ctx, &ConfigurationManager_ServiceDesc.Streams[0], ConfigurationManager_WatchTrench_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &configurationManagerWatchTrenchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ConfigurationManager_WatchTrenchClient interface {
	Recv() (*TrenchResponse, error)
	grpc.ClientStream
}

type configurationManagerWatchTrenchClient struct {
	grpc.ClientStream
}

func (x *configurationManagerWatchTrenchClient) Recv() (*TrenchResponse, error) {
	m := new(TrenchResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configurationManagerClient) WatchConduit(ctx context.Context, in *Conduit, opts ...grpc.CallOption) (ConfigurationManager_WatchConduitClient, error) {
	stream, err := c.cc.NewStream(ctx, &ConfigurationManager_ServiceDesc.Streams[1], ConfigurationManager_WatchConduit_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &configurationManagerWatchConduitClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ConfigurationManager_WatchConduitClient interface {
	Recv() (*ConduitResponse, error)
	grpc.ClientStream
}

type configurationManagerWatchConduitClient struct {
	grpc.ClientStream
}

func (x *configurationManagerWatchConduitClient) Recv() (*ConduitResponse, error) {
	m := new(ConduitResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configurationManagerClient) WatchStream(ctx context.Context, in *Stream, opts ...grpc.CallOption) (ConfigurationManager_WatchStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &ConfigurationManager_ServiceDesc.Streams[2], ConfigurationManager_WatchStream_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &configurationManagerWatchStreamClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ConfigurationManager_WatchStreamClient interface {
	Recv() (*StreamResponse, error)
	grpc.ClientStream
}

type configurationManagerWatchStreamClient struct {
	grpc.ClientStream
}

func (x *configurationManagerWatchStreamClient) Recv() (*StreamResponse, error) {
	m := new(StreamResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configurationManagerClient) WatchFlow(ctx context.Context, in *Flow, opts ...grpc.CallOption) (ConfigurationManager_WatchFlowClient, error) {
	stream, err := c.cc.NewStream(ctx, &ConfigurationManager_ServiceDesc.Streams[3], ConfigurationManager_WatchFlow_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &configurationManagerWatchFlowClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ConfigurationManager_WatchFlowClient interface {
	Recv() (*FlowResponse, error)
	grpc.ClientStream
}

type configurationManagerWatchFlowClient struct {
	grpc.ClientStream
}

func (x *configurationManagerWatchFlowClient) Recv() (*FlowResponse, error) {
	m := new(FlowResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configurationManagerClient) WatchVip(ctx context.Context, in *Vip, opts ...grpc.CallOption) (ConfigurationManager_WatchVipClient, error) {
	stream, err := c.cc.NewStream(ctx, &ConfigurationManager_ServiceDesc.Streams[4], ConfigurationManager_WatchVip_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &configurationManagerWatchVipClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ConfigurationManager_WatchVipClient interface {
	Recv() (*VipResponse, error)
	grpc.ClientStream
}

type configurationManagerWatchVipClient struct {
	grpc.ClientStream
}

func (x *configurationManagerWatchVipClient) Recv() (*VipResponse, error) {
	m := new(VipResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configurationManagerClient) WatchAttractor(ctx context.Context, in *Attractor, opts ...grpc.CallOption) (ConfigurationManager_WatchAttractorClient, error) {
	stream, err := c.cc.NewStream(ctx, &ConfigurationManager_ServiceDesc.Streams[5], ConfigurationManager_WatchAttractor_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &configurationManagerWatchAttractorClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ConfigurationManager_WatchAttractorClient interface {
	Recv() (*AttractorResponse, error)
	grpc.ClientStream
}

type configurationManagerWatchAttractorClient struct {
	grpc.ClientStream
}

func (x *configurationManagerWatchAttractorClient) Recv() (*AttractorResponse, error) {
	m := new(AttractorResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configurationManagerClient) WatchGateway(ctx context.Context, in *Gateway, opts ...grpc.CallOption) (ConfigurationManager_WatchGatewayClient, error) {
	stream, err := c.cc.NewStream(ctx, &ConfigurationManager_ServiceDesc.Streams[6], ConfigurationManager_WatchGateway_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &configurationManagerWatchGatewayClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ConfigurationManager_WatchGatewayClient interface {
	Recv() (*GatewayResponse, error)
	grpc.ClientStream
}

type configurationManagerWatchGatewayClient struct {
	grpc.ClientStream
}

func (x *configurationManagerWatchGatewayClient) Recv() (*GatewayResponse, error) {
	m := new(GatewayResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ConfigurationManagerServer is the server API for ConfigurationManager service.
// All implementations must embed UnimplementedConfigurationManagerServer
// for forward compatibility
type ConfigurationManagerServer interface {
	WatchTrench(*Trench, ConfigurationManager_WatchTrenchServer) error
	WatchConduit(*Conduit, ConfigurationManager_WatchConduitServer) error
	WatchStream(*Stream, ConfigurationManager_WatchStreamServer) error
	WatchFlow(*Flow, ConfigurationManager_WatchFlowServer) error
	WatchVip(*Vip, ConfigurationManager_WatchVipServer) error
	WatchAttractor(*Attractor, ConfigurationManager_WatchAttractorServer) error
	WatchGateway(*Gateway, ConfigurationManager_WatchGatewayServer) error
	mustEmbedUnimplementedConfigurationManagerServer()
}

// UnimplementedConfigurationManagerServer must be embedded to have forward compatible implementations.
type UnimplementedConfigurationManagerServer struct {
}

func (UnimplementedConfigurationManagerServer) WatchTrench(*Trench, ConfigurationManager_WatchTrenchServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchTrench not implemented")
}
func (UnimplementedConfigurationManagerServer) WatchConduit(*Conduit, ConfigurationManager_WatchConduitServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchConduit not implemented")
}
func (UnimplementedConfigurationManagerServer) WatchStream(*Stream, ConfigurationManager_WatchStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchStream not implemented")
}
func (UnimplementedConfigurationManagerServer) WatchFlow(*Flow, ConfigurationManager_WatchFlowServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchFlow not implemented")
}
func (UnimplementedConfigurationManagerServer) WatchVip(*Vip, ConfigurationManager_WatchVipServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchVip not implemented")
}
func (UnimplementedConfigurationManagerServer) WatchAttractor(*Attractor, ConfigurationManager_WatchAttractorServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchAttractor not implemented")
}
func (UnimplementedConfigurationManagerServer) WatchGateway(*Gateway, ConfigurationManager_WatchGatewayServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchGateway not implemented")
}
func (UnimplementedConfigurationManagerServer) mustEmbedUnimplementedConfigurationManagerServer() {}

// UnsafeConfigurationManagerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ConfigurationManagerServer will
// result in compilation errors.
type UnsafeConfigurationManagerServer interface {
	mustEmbedUnimplementedConfigurationManagerServer()
}

func RegisterConfigurationManagerServer(s grpc.ServiceRegistrar, srv ConfigurationManagerServer) {
	s.RegisterService(&ConfigurationManager_ServiceDesc, srv)
}

func _ConfigurationManager_WatchTrench_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Trench)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfigurationManagerServer).WatchTrench(m, &configurationManagerWatchTrenchServer{stream})
}

type ConfigurationManager_WatchTrenchServer interface {
	Send(*TrenchResponse) error
	grpc.ServerStream
}

type configurationManagerWatchTrenchServer struct {
	grpc.ServerStream
}

func (x *configurationManagerWatchTrenchServer) Send(m *TrenchResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _ConfigurationManager_WatchConduit_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Conduit)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfigurationManagerServer).WatchConduit(m, &configurationManagerWatchConduitServer{stream})
}

type ConfigurationManager_WatchConduitServer interface {
	Send(*ConduitResponse) error
	grpc.ServerStream
}

type configurationManagerWatchConduitServer struct {
	grpc.ServerStream
}

func (x *configurationManagerWatchConduitServer) Send(m *ConduitResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _ConfigurationManager_WatchStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Stream)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfigurationManagerServer).WatchStream(m, &configurationManagerWatchStreamServer{stream})
}

type ConfigurationManager_WatchStreamServer interface {
	Send(*StreamResponse) error
	grpc.ServerStream
}

type configurationManagerWatchStreamServer struct {
	grpc.ServerStream
}

func (x *configurationManagerWatchStreamServer) Send(m *StreamResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _ConfigurationManager_WatchFlow_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Flow)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfigurationManagerServer).WatchFlow(m, &configurationManagerWatchFlowServer{stream})
}

type ConfigurationManager_WatchFlowServer interface {
	Send(*FlowResponse) error
	grpc.ServerStream
}

type configurationManagerWatchFlowServer struct {
	grpc.ServerStream
}

func (x *configurationManagerWatchFlowServer) Send(m *FlowResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _ConfigurationManager_WatchVip_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Vip)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfigurationManagerServer).WatchVip(m, &configurationManagerWatchVipServer{stream})
}

type ConfigurationManager_WatchVipServer interface {
	Send(*VipResponse) error
	grpc.ServerStream
}

type configurationManagerWatchVipServer struct {
	grpc.ServerStream
}

func (x *configurationManagerWatchVipServer) Send(m *VipResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _ConfigurationManager_WatchAttractor_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Attractor)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfigurationManagerServer).WatchAttractor(m, &configurationManagerWatchAttractorServer{stream})
}

type ConfigurationManager_WatchAttractorServer interface {
	Send(*AttractorResponse) error
	grpc.ServerStream
}

type configurationManagerWatchAttractorServer struct {
	grpc.ServerStream
}

func (x *configurationManagerWatchAttractorServer) Send(m *AttractorResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _ConfigurationManager_WatchGateway_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Gateway)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConfigurationManagerServer).WatchGateway(m, &configurationManagerWatchGatewayServer{stream})
}

type ConfigurationManager_WatchGatewayServer interface {
	Send(*GatewayResponse) error
	grpc.ServerStream
}

type configurationManagerWatchGatewayServer struct {
	grpc.ServerStream
}

func (x *configurationManagerWatchGatewayServer) Send(m *GatewayResponse) error {
	return x.ServerStream.SendMsg(m)
}

// ConfigurationManager_ServiceDesc is the grpc.ServiceDesc for ConfigurationManager service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ConfigurationManager_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "nsp.v1.ConfigurationManager",
	HandlerType: (*ConfigurationManagerServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "WatchTrench",
			Handler:       _ConfigurationManager_WatchTrench_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "WatchConduit",
			Handler:       _ConfigurationManager_WatchConduit_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "WatchStream",
			Handler:       _ConfigurationManager_WatchStream_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "WatchFlow",
			Handler:       _ConfigurationManager_WatchFlow_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "WatchVip",
			Handler:       _ConfigurationManager_WatchVip_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "WatchAttractor",
			Handler:       _ConfigurationManager_WatchAttractor_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "WatchGateway",
			Handler:       _ConfigurationManager_WatchGateway_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/nsp/v1/configurationmanager.proto",
}
