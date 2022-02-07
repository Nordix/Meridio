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
	"github.com/nordix/meridio/pkg/ambassador/tap/stream"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	typesMocks "github.com/nordix/meridio/pkg/ambassador/tap/types/mocks"
	nsmMocks "github.com/nordix/meridio/pkg/nsm/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Test_Constructor(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	logrus.SetLevel(logrus.FatalLevel)

	c := &ambassadorAPI.Conduit{
		Name: "conduit-a",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	targetName := "abc"
	namespace := "red"
	node := "worker"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)

	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
}

func Test_Connect_Disconnect(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	logrus.SetLevel(logrus.FatalLevel)

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
	configurationCtx, configurationCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	configuration.EXPECT().WatchVIPs(gomock.Any()).DoAndReturn(func(ctx context.Context) {
		<-ctx.Done()
		configurationCancel()
	})
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

	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.Configuration = configuration

	err = cndt.Connect(context.TODO())
	assert.Nil(t, err)
	assert.Equal(t, srcIpAddrs, cndt.GetIPs())

	err = cndt.Disconnect(context.TODO())
	assert.Nil(t, err)

	<-configurationCtx.Done()
}

func Test_AddStream_While_Disconnected(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	logrus.SetLevel(logrus.FatalLevel)

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
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)
	networkServiceClient.EXPECT().Request(gomock.Any(), gomock.Any()).Return(nil, nil)
	streamFactory := mocks.NewMockStreamFactory(ctrl)
	configuration := mocks.NewMockConfiguration(ctrl)
	configuration.EXPECT().WatchVIPs(gomock.Any()).Return().AnyTimes()
	streamA := typesMocks.NewMockStream(ctrl)

	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.StreamFactory = streamFactory
	cndt.Configuration = configuration

	streamFactory.EXPECT().New(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, c stream.Conduit) (types.Stream, error) {
		assert.Equal(t, cndt, c)
		assert.Equal(t, s, strm)
		return streamA, nil
	})

	// AddStream on disconnected Conduit
	strm, err := cndt.AddStream(context.TODO(), s)
	assert.Nil(t, err)
	assert.Equal(t, streamA, strm)
	streams := cndt.GetStreams()
	assert.Len(t, streams, 1)
	assert.Contains(t, streams, strm)

	openCtx, openCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	streamA.EXPECT().Open(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		openCancel()
		return nil
	})
	err = cndt.Connect(context.TODO())
	assert.Nil(t, err)

	<-openCtx.Done()

	streamA.EXPECT().Close(gomock.Any()).Return(nil)
	err = cndt.Disconnect(context.TODO())
	assert.Nil(t, err)
}

func Test_AddStream_While_Connected(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	logrus.SetLevel(logrus.FatalLevel)

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
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)
	networkServiceClient.EXPECT().Request(gomock.Any(), gomock.Any()).Return(nil, nil)
	streamFactory := mocks.NewMockStreamFactory(ctrl)
	configuration := mocks.NewMockConfiguration(ctrl)
	configuration.EXPECT().WatchVIPs(gomock.Any()).Return().AnyTimes()
	streamA := typesMocks.NewMockStream(ctrl)

	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.StreamFactory = streamFactory
	cndt.Configuration = configuration

	streamFactory.EXPECT().New(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, c stream.Conduit) (types.Stream, error) {
		assert.Equal(t, cndt, c)
		assert.Equal(t, s, strm)
		return streamA, nil
	})

	err = cndt.Connect(context.TODO())
	assert.Nil(t, err)

	openCtx, openCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	streamA.EXPECT().Open(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		openCancel()
		return nil
	})

	// AddStream on disconnected Conduit
	strm, err := cndt.AddStream(context.TODO(), s)
	assert.Nil(t, err)
	assert.Equal(t, streamA, strm)
	streams := cndt.GetStreams()
	assert.Len(t, streams, 1)
	assert.Contains(t, streams, strm)

	<-openCtx.Done()

	streamA.EXPECT().Close(gomock.Any()).Return(nil)
	err = cndt.Disconnect(context.TODO())
	assert.Nil(t, err)
}

func Test_AddStream_Invalid(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	logrus.SetLevel(logrus.FatalLevel)

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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)
	streamFactory := mocks.NewMockStreamFactory(ctrl)

	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.StreamFactory = streamFactory

	strm, err := cndt.AddStream(context.TODO(), s)
	assert.NotNil(t, err)
	assert.Nil(t, strm)
}

func Test_AddStream_Existing(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	logrus.SetLevel(logrus.FatalLevel)

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
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)
	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamA.EXPECT().Equals(gomock.Any()).Return(true)

	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.StreamFactory = streamFactory

	streamFactory.EXPECT().New(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, c stream.Conduit) (types.Stream, error) {
		assert.Equal(t, cndt, c)
		assert.Equal(t, s, strm)
		return streamA, nil
	})

	// AddStream
	strm, err := cndt.AddStream(context.TODO(), s)
	assert.Nil(t, err)
	assert.NotNil(t, strm)
	// Re-Add existing one
	ns, err := cndt.AddStream(context.TODO(), s)
	assert.Nil(t, err)
	assert.NotNil(t, ns)
	streams := cndt.GetStreams()
	assert.Len(t, streams, 1)
	assert.Contains(t, streams, strm)
}

func Test_RemoveStream_While_Disconnected(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	logrus.SetLevel(logrus.FatalLevel)

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
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)
	networkServiceClient.EXPECT().Request(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	streamFactory := mocks.NewMockStreamFactory(ctrl)
	configuration := mocks.NewMockConfiguration(ctrl)
	configuration.EXPECT().WatchVIPs(gomock.Any()).Return().AnyTimes()
	streamA := typesMocks.NewMockStream(ctrl)
	streamA.EXPECT().Equals(gomock.Any()).Return(true).AnyTimes()
	streamA.EXPECT().Close(gomock.Any()).Return(nil)
	streamRegistry := typesMocks.NewMockRegistry(ctrl)
	streamRegistry.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)

	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, streamRegistry, nil)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.StreamFactory = streamFactory
	cndt.Configuration = configuration

	streamFactory.EXPECT().New(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, c stream.Conduit) (types.Stream, error) {
		assert.Equal(t, cndt, c)
		assert.Equal(t, s, strm)
		return streamA, nil
	})

	// AddStream on disconnected Conduit
	_, err = cndt.AddStream(context.TODO(), s)
	assert.Nil(t, err)
	err = cndt.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)
	streams := cndt.GetStreams()
	assert.Len(t, streams, 0)

	err = cndt.Connect(context.TODO())
	assert.Nil(t, err)

	err = cndt.Disconnect(context.TODO())
	assert.Nil(t, err)
}

func Test_RemoveStream_While_Connected(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	logrus.SetLevel(logrus.FatalLevel)

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
	networkServiceClient := nsmMocks.NewMockNetworkServiceClient(ctrl)
	networkServiceClient.EXPECT().Request(gomock.Any(), gomock.Any()).Return(nil, nil)
	streamFactory := mocks.NewMockStreamFactory(ctrl)
	configuration := mocks.NewMockConfiguration(ctrl)
	configuration.EXPECT().WatchVIPs(gomock.Any()).Return().AnyTimes()
	streamA := typesMocks.NewMockStream(ctrl)
	streamA.EXPECT().Equals(gomock.Any()).Return(true).AnyTimes()
	streamRegistry := typesMocks.NewMockRegistry(ctrl)
	streamRegistry.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)

	cndt, err := conduit.New(c, targetName, namespace, node, nil, nil, networkServiceClient, streamRegistry, nil)
	assert.Nil(t, err)
	assert.NotNil(t, cndt)
	cndt.StreamFactory = streamFactory
	cndt.Configuration = configuration

	streamFactory.EXPECT().New(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, c stream.Conduit) (types.Stream, error) {
		assert.Equal(t, cndt, c)
		assert.Equal(t, s, strm)
		return streamA, nil
	})

	err = cndt.Connect(context.TODO())
	assert.Nil(t, err)

	streamA.EXPECT().Open(gomock.Any()).Return(nil).AnyTimes()
	streamA.EXPECT().Close(gomock.Any()).Return(nil)

	strm, err := cndt.AddStream(context.TODO(), s)
	assert.Nil(t, err)
	assert.Equal(t, streamA, strm)

	err = cndt.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)
	streams := cndt.GetStreams()
	assert.Len(t, streams, 0)

	err = cndt.Disconnect(context.TODO())
	assert.Nil(t, err)
}
