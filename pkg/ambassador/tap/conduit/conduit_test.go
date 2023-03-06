/*
Copyright (c) 2021-2023 Nordix Foundation

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

package conduit_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/conduit"
	"github.com/nordix/meridio/pkg/ambassador/tap/conduit/mocks"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream/registry"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	typesMocks "github.com/nordix/meridio/pkg/ambassador/tap/types/mocks"
	nsmMocks "github.com/nordix/meridio/pkg/nsm/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Test_Connect_Disconnect(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	c := &ambassadorAPI.Conduit{
		Name: "conduit-a",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	targetName := "abc"
	namespace := "red"
	node := "worker"
	srcIpAddrs := []string{"172.16.0.1/24", "fd00::1/64"}
	dstIpAddrs := []string{"172.16.0.2/24", "fd00::2/64"}
	id := ""

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	configuration := mocks.NewMockConfiguration(ctrl)
	configuration.EXPECT().Watch().Return()
	configuration.EXPECT().Stop().Return()
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)
	networkServiceClient.EXPECT().Request(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
		assert.NotNil(t, in)
		assert.NotNil(t, in.GetConnection())
		id = in.GetConnection().GetId()
		in.GetConnection().Context = &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				SrcIpAddrs: srcIpAddrs,
				DstIpAddrs: dstIpAddrs,
			},
		}
		return in.GetConnection(), nil
	})
	networkServiceClient.EXPECT().Close(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
		assert.NotNil(t, in)
		assert.Equal(t, id, in.GetId())
		return nil, nil
	})

	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, nil, nil, nil, 30*time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.Configuration = configuration

	assert.Len(t, cndt.GetIPs(), 0)

	err = cndt.Connect(context.TODO())
	assert.Nil(t, err)
	assert.Equal(t, srcIpAddrs, cndt.GetIPs())

	err = cndt.Disconnect(context.TODO())
	assert.Nil(t, err)
	assert.Len(t, cndt.GetIPs(), 0)
}

func Test_AddStream(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	s := &ambassadorAPI.Stream{
		Name: "stream-a",
		Conduit: &ambassadorAPI.Conduit{
			Name: "conduit-a",
			Trench: &ambassadorAPI.Trench{
				Name: "trench-a",
			},
		},
	}
	c := s.Conduit
	targetName := "abc"
	namespace := "red"
	node := "worker"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cndt, _ := conduit.New(c, targetName, namespace, node, nil, nil, nil, nil, nil, nil, 30*time.Second)
	streamRegistry := registry.New()
	cndt.StreamManager = conduit.NewStreamManager(nil, nil, streamRegistry, fakeStreamFactory(ctrl), 0, 30*time.Second)

	err := cndt.AddStream(context.TODO(), s)
	assert.Nil(t, err)

	err = cndt.AddStream(context.TODO(), s)
	assert.Nil(t, err)

	streams := cndt.GetStreams()
	assert.NotNil(t, streams)
	assert.Len(t, streams, 1)
	assert.Contains(t, streams, s)
}

func Test_AddStream_Invalid(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	s := &ambassadorAPI.Stream{
		Name: "stream-a",
		Conduit: &ambassadorAPI.Conduit{
			Name: "conduit-a",
			Trench: &ambassadorAPI.Trench{
				Name: "trench-a",
			},
		},
	}
	c := &ambassadorAPI.Conduit{
		Name: "conduit-b",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	targetName := "abc"
	namespace := "red"
	node := "worker"

	cndt, _ := conduit.New(c, targetName, namespace, node, nil, nil, nil, nil, nil, nil, 30*time.Second)

	err := cndt.AddStream(context.TODO(), s)
	assert.NotNil(t, err)
}

func Test_RemoveStream(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	c := &ambassadorAPI.Conduit{
		Name: "conduit-a",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	s1 := &ambassadorAPI.Stream{
		Name:    "stream-a",
		Conduit: c,
	}
	s2 := &ambassadorAPI.Stream{
		Name:    "stream-b",
		Conduit: c,
	}
	targetName := "abc"
	namespace := "red"
	node := "worker"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cndt, _ := conduit.New(c, targetName, namespace, node, nil, nil, nil, nil, nil, nil, 30*time.Second)
	streamRegistry := registry.New()
	cndt.StreamManager = conduit.NewStreamManager(nil, nil, streamRegistry, fakeStreamFactory(ctrl), 0, 30*time.Second)

	err := cndt.AddStream(context.TODO(), s2)
	assert.Nil(t, err)

	err = cndt.AddStream(context.TODO(), s1)
	assert.Nil(t, err)

	streams := cndt.GetStreams()
	assert.NotNil(t, streams)
	assert.Len(t, streams, 2)
	assert.Contains(t, streams, s1)
	assert.Contains(t, streams, s2)

	err = cndt.RemoveStream(context.TODO(), s1)
	assert.Nil(t, err)

	streams = cndt.GetStreams()
	assert.NotNil(t, streams)
	assert.Len(t, streams, 1)
	assert.Contains(t, streams, s2)

	err = cndt.RemoveStream(context.TODO(), s2)
	assert.Nil(t, err)

	streams = cndt.GetStreams()
	assert.NotNil(t, streams)
	assert.Len(t, streams, 0)
}

func Test_SetVIPs_Not_Connected(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	c := &ambassadorAPI.Conduit{
		Name: "conduit-a",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	targetName := "abc"
	namespace := "red"
	node := "worker"
	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, nil, nil, nil, nil, 30*time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)

	err = cndt.SetVIPs(context.Background(), []string{})
	assert.Nil(t, err)
}

// 1. Create the conduit
// 2. Connect the conduit
// 3. Verify Request has been called (and set IPContext)
// 4. verify local IPs
// 5. Call SetVips
// 6. Verify Request has been called and verify IPContext
// 7. verify local IPs
// 8. Disconnect the conduit
// 9. Verify Close has been called
func Test_SetVIPs(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	c := &ambassadorAPI.Conduit{
		Name: "conduit-a",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	targetName := "abc"
	namespace := "red"
	node := "worker"
	vips := []string{"20.0.0.1/32", "2000::1/128"}
	srcIpAddrs := []string{"172.16.0.1/24", "fd00::1/64"}
	dstIpAddrs := []string{"172.16.0.2/24", "fd00::2/64"}
	extraPrefixes := []string{"172.16.0.100/24", "fd00::100/64"}
	id := ""

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	configuration := mocks.NewMockConfiguration(ctrl)
	configuration.EXPECT().Watch().Return()
	configuration.EXPECT().Stop().Return()
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)
	// 3.
	firstRequest := networkServiceClient.EXPECT().Request(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
		assert.NotNil(t, in)
		assert.NotNil(t, in.GetConnection())
		id = in.GetConnection().GetId()
		in.GetConnection().Context = &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				SrcIpAddrs:    srcIpAddrs,
				DstIpAddrs:    dstIpAddrs,
				ExtraPrefixes: extraPrefixes,
			},
		}
		return in.GetConnection(), nil
	})
	// 6.
	networkServiceClient.EXPECT().Request(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
		assert.NotNil(t, in)
		assert.NotNil(t, in.GetConnection())
		assert.NotNil(t, in.GetConnection().GetContext())
		ipContext := in.GetConnection().GetContext().GetIpContext()
		assert.NotNil(t, ipContext)
		assert.ElementsMatch(t, dstIpAddrs, ipContext.DstIpAddrs)
		assert.ElementsMatch(t, append(srcIpAddrs, vips...), ipContext.SrcIpAddrs)
		assert.Len(t, ipContext.Policies, 2)
		assert.ElementsMatch(t, []*networkservice.PolicyRoute{
			{
				From: "20.0.0.1/32",
				Routes: []*networkservice.Route{
					{
						Prefix:  "0.0.0.0/0",
						NextHop: "172.16.0.100",
					},
				},
			},
			{
				From: "2000::1/128",
				Routes: []*networkservice.Route{
					{
						Prefix:  "::/0",
						NextHop: "fd00::100",
					},
				},
			},
		}, ipContext.Policies)
		return in.GetConnection(), nil
	}).After(firstRequest)
	// 9.
	networkServiceClient.EXPECT().Close(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
		assert.NotNil(t, in)
		assert.Equal(t, id, in.GetId())
		return nil, nil
	})

	// 1.
	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, nil, nil, nil, 30*time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.Configuration = configuration

	// 2.
	err = cndt.Connect(context.TODO())
	assert.Nil(t, err)

	// 4.
	assert.Equal(t, srcIpAddrs, cndt.GetIPs())

	// 5.
	err = cndt.SetVIPs(context.Background(), vips)
	assert.Nil(t, err)

	// 7.
	assert.Equal(t, srcIpAddrs, cndt.GetIPs())

	// 8.
	err = cndt.Disconnect(context.TODO())
	assert.Nil(t, err)
	assert.Len(t, cndt.GetIPs(), 0)
}

// 1. Create the conduit
// 2. Connect the conduit
// 3. Verify Request has been called (and set IPContext)
// 4. verify local IPs
// 5. Call SetVips
// 6. Verify Request has been called and change the IPs
// 7. verify local IPs are the new IPs
// 8. Disconnect the conduit
// 9. Verify Close has been called
func Test_LocalIPs_Switch(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	c := &ambassadorAPI.Conduit{
		Name: "conduit-a",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	targetName := "abc"
	namespace := "red"
	node := "worker"
	vips := []string{"20.0.0.1/32", "2000::1/128"}
	srcIpAddrs := []string{"172.16.0.1/24", "fd00::1/64"}
	dstIpAddrs := []string{"172.16.0.2/24", "fd00::2/64"}
	extraPrefixes := []string{"172.16.0.100/24", "fd00::100/64"}

	srcIpAddrs2 := []string{"172.16.0.10/24", "fd00::10/64"}
	dstIpAddrs2 := []string{"172.16.0.20/24", "fd00::20/64"}
	extraPrefixes2 := []string{"172.16.0.200/24", "fd00::200/64"}
	id := ""

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	configuration := mocks.NewMockConfiguration(ctrl)
	configuration.EXPECT().Watch().Return()
	configuration.EXPECT().Stop().Return()
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)
	// 3.
	firstRequest := networkServiceClient.EXPECT().Request(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
		assert.NotNil(t, in)
		assert.NotNil(t, in.GetConnection())
		id = in.GetConnection().GetId()
		in.GetConnection().Context = &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				SrcIpAddrs:    srcIpAddrs,
				DstIpAddrs:    dstIpAddrs,
				ExtraPrefixes: extraPrefixes,
			},
		}
		return in.GetConnection(), nil
	})
	// 6.
	networkServiceClient.EXPECT().Request(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
		assert.NotNil(t, in)
		assert.NotNil(t, in.GetConnection())
		assert.NotNil(t, in.GetConnection().GetContext())
		ipContext := in.GetConnection().GetContext().GetIpContext()
		assert.NotNil(t, ipContext)
		assert.ElementsMatch(t, dstIpAddrs, ipContext.DstIpAddrs)
		assert.ElementsMatch(t, append(srcIpAddrs, vips...), ipContext.SrcIpAddrs)
		assert.Len(t, ipContext.Policies, 2)
		ipContext.DstIpAddrs = dstIpAddrs2
		ipContext.SrcIpAddrs = srcIpAddrs2
		ipContext.ExtraPrefixes = extraPrefixes2
		return in.GetConnection(), nil
	}).After(firstRequest)
	// 9.
	networkServiceClient.EXPECT().Close(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
		assert.NotNil(t, in)
		assert.Equal(t, id, in.GetId())
		return nil, nil
	})

	// 1.
	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, nil, nil, nil, 30*time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.Configuration = configuration

	// 2.
	err = cndt.Connect(context.TODO())
	assert.Nil(t, err)

	// 4.
	assert.Equal(t, srcIpAddrs, cndt.GetIPs())

	// 5.
	err = cndt.SetVIPs(context.Background(), vips)
	assert.Nil(t, err)

	// 7.
	assert.Equal(t, srcIpAddrs2, cndt.GetIPs())

	// 8.
	err = cndt.Disconnect(context.TODO())
	assert.Nil(t, err)
	assert.Len(t, cndt.GetIPs(), 0)
}

func fakeStreamFactory(ctrl *gomock.Controller) *mocks.MockStreamFactory {
	factory := mocks.NewMockStreamFactory(ctrl)
	factory.EXPECT().New(gomock.Any()).DoAndReturn(func(s *ambassadorAPI.Stream) (types.Stream, error) {
		newStream := typesMocks.NewMockStream(ctrl)
		newStream.EXPECT().Open(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		newStream.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()
		newStream.EXPECT().Equals(gomock.Any()).DoAndReturn(func(s1 *ambassadorAPI.Stream) bool {
			return s1.Equals(s)
		}).AnyTimes()
		newStream.EXPECT().GetStream().Return(s).AnyTimes()
		return newStream, nil
	}).AnyTimes()
	return factory
}
