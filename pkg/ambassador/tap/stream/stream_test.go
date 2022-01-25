/*
Copyright (c) 2021 Nordix Foundation

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

package stream_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	nspMock "github.com/nordix/meridio/api/nsp/v1/mocks"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream"
	"github.com/nordix/meridio/pkg/ambassador/tap/types/mocks"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Test_GetName(t *testing.T) {
	name := "test-stream"
	s := &stream.Stream{
		Name: name,
	}
	assert.Equal(t, s.GetName(), name)
}

func Test_Request(t *testing.T) {
	// Registers with a unused identifier as disabled target
	// Checks there is no collision
	// Update the target as enabled
	// Sends event the target is registered and enabled
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	trenchName := "test-trench"
	conduitName := "test-conduit"
	maxNumberOfTargets := 2
	identifierSelected := "0"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt, _, nspClient, targetRegistryWatchClient := getConduitTrenchNSP(ctrl, trenchName, conduitName, ips)
	firstGetCall := targetRegistryWatchClient.EXPECT().Recv().Return(getTargetsResponse([]string{"1"}), nil)
	targetRegistryWatchClient.EXPECT().Recv().Return(getTargetsResponse([]string{"1", identifierSelected}), nil).After(firstGetCall)
	nspClient.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target, _ ...grpc.CallOption) (*empty.Empty, error) {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Status, nspAPI.Target_DISABLED)
		identifier, exists := target.Context[types.IdentifierKey]
		assert.True(t, exists)
		assert.NotEqual(t, identifier, "1")
		identifierInt, err := strconv.Atoi(identifier)
		assert.Nil(t, err)
		assert.Greater(t, identifierInt, 0)
		assert.LessOrEqual(t, identifierInt, maxNumberOfTargets)
		identifierSelected = identifier
		return &emptypb.Empty{}, nil
	})
	nspClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target, _ ...grpc.CallOption) (*empty.Empty, error) {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Status, nspAPI.Target_ENABLED)
		identifier, exists := target.Context[types.IdentifierKey]
		assert.True(t, exists)
		assert.Equal(t, identifier, identifierSelected)
		return &emptypb.Empty{}, nil
	})

	eventChan := make(chan struct{}, 10)
	s := &stream.Stream{
		Conduit:            cndt,
		EventChan:          eventChan,
		MaxNumberOfTargets: maxNumberOfTargets,
	}
	err := s.Open(context.Background())
	assert.Nil(t, err)
	var streamEvent interface{}
	select {
	case streamEvent = <-eventChan:
	default:
	}
	assert.NotNil(t, streamEvent)
}

func Test_Request_NoIdentifierAvailable(t *testing.T) {
	// Detects no identifier is available
	// Returns an error
	// Does not send any event
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	trenchName := "test-trench"
	conduitName := "test-conduit"
	maxNumberOfTargets := 2
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt, _, _, targetRegistryWatchClient := getConduitTrenchNSP(ctrl, trenchName, conduitName, ips)
	targetRegistryWatchClient.EXPECT().Recv().Return(getTargetsResponse([]string{"1", "2"}), nil)

	eventChan := make(chan struct{}, 10)
	s := &stream.Stream{
		Conduit:            cndt,
		EventChan:          eventChan,
		MaxNumberOfTargets: maxNumberOfTargets,
	}
	err := s.Open(context.Background())
	assert.NotNil(t, err)
	var streamEvent interface{}
	select {
	case streamEvent = <-eventChan:
	default:
	}
	fmt.Println(streamEvent)
	assert.Nil(t, streamEvent)
}

func Test_Request_NoNSPConnection(t *testing.T) {
	// Returns an error
	// Does not send any event
	trenchName := "test-trench"
	conduitName := "test-conduit"
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt, _, _, targetRegistryWatchClient := getConduitTrenchNSP(ctrl, trenchName, conduitName, ips)
	// nspClient.EXPECT().GetTargets(gomock.Any(), gomock.Any()).Return(nil, errors.New(""))
	targetRegistryWatchClient.EXPECT().Recv().Return(nil, errors.New(""))

	eventChan := make(chan struct{}, 10)
	s := &stream.Stream{
		Conduit:   cndt,
		EventChan: eventChan,
	}
	err := s.Open(context.Background())
	assert.NotNil(t, err)
	var streamEvent interface{}
	select {
	case streamEvent = <-eventChan:
	default:
	}
	assert.Nil(t, streamEvent)
}

func Test_Request_Concurrent(t *testing.T) {
	// Registers with a unused identifier as disabled target
	// Detects there is a collision
	// Updates with a new unused identifier still as disabled target
	// Checks there is no collision
	// Update the target as enabled
	// Sends event the target is registered and enabled
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	trenchName := "test-trench"
	conduitName := "test-conduit"
	maxNumberOfTargets := 3
	concurrentIdentifier := "0"
	identifierSelected := "0"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt, _, nspClient, targetRegistryWatchClient := getConduitTrenchNSP(ctrl, trenchName, conduitName, ips)
	firstGet := targetRegistryWatchClient.EXPECT().Recv().Return(getTargetsResponse([]string{"1"}), nil)
	secondGet := targetRegistryWatchClient.EXPECT().Recv().DoAndReturn(func() (*nspAPI.TargetResponse, error) {
		concurrentIdentifier = identifierSelected
		return getTargetsResponse([]string{"1", concurrentIdentifier, identifierSelected}), nil
	}).After(firstGet)
	targetRegistryWatchClient.EXPECT().Recv().Return(getTargetsResponse([]string{"1", concurrentIdentifier, identifierSelected}), nil).After(secondGet)
	nspClient.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target, _ ...grpc.CallOption) (*empty.Empty, error) {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Status, nspAPI.Target_DISABLED)
		identifier, exists := target.Context[types.IdentifierKey]
		assert.True(t, exists)
		assert.NotEqual(t, identifier, "1")
		identifierInt, err := strconv.Atoi(identifier)
		assert.Nil(t, err)
		assert.Greater(t, identifierInt, 0)
		assert.LessOrEqual(t, identifierInt, maxNumberOfTargets)
		identifierSelected = identifier
		return &emptypb.Empty{}, nil
	})
	firstUpdate := nspClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target, _ ...grpc.CallOption) (*empty.Empty, error) {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Status, nspAPI.Target_DISABLED)
		identifier, exists := target.Context[types.IdentifierKey]
		assert.True(t, exists)
		assert.NotEqual(t, identifier, "1")
		assert.NotEqual(t, identifier, concurrentIdentifier)
		identifierInt, err := strconv.Atoi(identifier)
		assert.Nil(t, err)
		assert.Greater(t, identifierInt, 0)
		assert.LessOrEqual(t, identifierInt, maxNumberOfTargets)
		identifierSelected = identifier
		return &emptypb.Empty{}, nil
	})
	nspClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target, _ ...grpc.CallOption) (*empty.Empty, error) {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Status, nspAPI.Target_ENABLED)
		identifier, exists := target.Context[types.IdentifierKey]
		assert.True(t, exists)
		assert.Equal(t, identifier, identifierSelected)
		return &emptypb.Empty{}, nil
	}).After(firstUpdate)

	eventChan := make(chan struct{}, 10)
	s := &stream.Stream{
		Conduit:            cndt,
		EventChan:          eventChan,
		MaxNumberOfTargets: maxNumberOfTargets,
	}
	err := s.Open(context.Background())
	assert.Nil(t, err)
	var streamEvent interface{}
	select {
	case streamEvent = <-eventChan:
	default:
	}
	assert.NotNil(t, streamEvent)
}

func Test_Request_Concurrent_NoIdentifierAvailable(t *testing.T) {
	// Registers with a unused identifier as disabled target
	// Detects there is a collision
	// Detects no identifier is available
	// Unregisters the target
	// Returns an error
	// Does not send any event
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	trenchName := "test-trench"
	conduitName := "test-conduit"
	maxNumberOfTargets := 2
	concurrentIdentifier := "0"
	identifierSelected := "0"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt, _, nspClient, targetRegistryWatchClient := getConduitTrenchNSP(ctrl, trenchName, conduitName, ips)
	firstGet := targetRegistryWatchClient.EXPECT().Recv().Return(getTargetsResponse([]string{"1"}), nil)
	targetRegistryWatchClient.EXPECT().Recv().DoAndReturn(func() (*nspAPI.TargetResponse, error) {
		concurrentIdentifier = identifierSelected
		return getTargetsResponse([]string{"1", concurrentIdentifier, identifierSelected}), nil
	}).After(firstGet)
	nspClient.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target, _ ...grpc.CallOption) (*empty.Empty, error) {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Status, nspAPI.Target_DISABLED)
		identifier, exists := target.Context[types.IdentifierKey]
		assert.True(t, exists)
		assert.NotEqual(t, identifier, "1")
		identifierInt, err := strconv.Atoi(identifier)
		assert.Nil(t, err)
		assert.Greater(t, identifierInt, 0)
		assert.LessOrEqual(t, identifierInt, maxNumberOfTargets)
		identifierSelected = identifier
		return &emptypb.Empty{}, nil
	})
	nspClient.EXPECT().Unregister(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target, _ ...grpc.CallOption) (*empty.Empty, error) {
		assert.NotNil(t, target)
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		return &emptypb.Empty{}, nil
	})

	eventChan := make(chan struct{}, 10)
	s := &stream.Stream{
		Conduit:            cndt,
		EventChan:          eventChan,
		MaxNumberOfTargets: maxNumberOfTargets,
	}
	err := s.Open(context.Background())
	assert.NotNil(t, err)
	var streamEvent interface{}
	select {
	case streamEvent = <-eventChan:
	default:
	}
	assert.Nil(t, streamEvent)
}

func Test_Request_Concurrent_NoIdentifierAvailable_NoNSPConnection(t *testing.T) {
	// Registers with a unused identifier as disabled target
	// Detects there is a collision
	// Detects no identifier is available
	// Can't unregister the target
	// Returns an error
	// Does not send any event
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	trenchName := "test-trench"
	conduitName := "test-conduit"
	maxNumberOfTargets := 2
	concurrentIdentifier := "0"
	identifierSelected := "0"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt, _, nspClient, targetRegistryWatchClient := getConduitTrenchNSP(ctrl, trenchName, conduitName, ips)
	firstGet := targetRegistryWatchClient.EXPECT().Recv().Return(getTargetsResponse([]string{"1"}), nil)
	targetRegistryWatchClient.EXPECT().Recv().DoAndReturn(func() (*nspAPI.TargetResponse, error) {
		concurrentIdentifier = identifierSelected
		return getTargetsResponse([]string{"1", concurrentIdentifier, identifierSelected}), nil
	}).After(firstGet)
	nspClient.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target, _ ...grpc.CallOption) (*empty.Empty, error) {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Status, nspAPI.Target_DISABLED)
		identifier, exists := target.Context[types.IdentifierKey]
		assert.True(t, exists)
		assert.NotEqual(t, identifier, "1")
		identifierInt, err := strconv.Atoi(identifier)
		assert.Nil(t, err)
		assert.Greater(t, identifierInt, 0)
		assert.LessOrEqual(t, identifierInt, maxNumberOfTargets)
		identifierSelected = identifier
		return &emptypb.Empty{}, nil
	})
	nspClient.EXPECT().Unregister(gomock.Any(), gomock.Any()).Return(&emptypb.Empty{}, errors.New(""))

	eventChan := make(chan struct{}, 10)
	s := &stream.Stream{
		Conduit:            cndt,
		EventChan:          eventChan,
		MaxNumberOfTargets: maxNumberOfTargets,
	}
	err := s.Open(context.Background())
	assert.NotNil(t, err)
	var streamEvent interface{}
	select {
	case streamEvent = <-eventChan:
	default:
	}
	assert.Nil(t, streamEvent)
}

func Test_Close(t *testing.T) {
	// Closes the connection with the correct IPs
	// Sends an event the target is now unregistered
	trenchName := "test-trench"
	conduitName := "test-conduit"
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt, _, nspClient, _ := getConduitTrenchNSP(ctrl, trenchName, conduitName, ips)
	nspClient.EXPECT().Unregister(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target, _ ...grpc.CallOption) (*empty.Empty, error) {
		assert.NotNil(t, target)
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		return &emptypb.Empty{}, nil
	})

	eventChan := make(chan struct{}, 10)
	s := &stream.Stream{
		Conduit:   cndt,
		EventChan: eventChan,
	}
	err := s.Close(context.Background())
	assert.Nil(t, err)
	var streamEvent interface{}
	select {
	case streamEvent = <-eventChan:
	default:
	}
	assert.NotNil(t, streamEvent)
}

func Test_Close_NoNSPConnection(t *testing.T) {
	// Returns an error
	// Does not send any event
	trenchName := "test-trench"
	conduitName := "test-conduit"
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt, _, nspClient, _ := getConduitTrenchNSP(ctrl, trenchName, conduitName, ips)
	nspClient.EXPECT().Unregister(gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

	eventChan := make(chan struct{}, 10)
	s := &stream.Stream{
		Conduit:   cndt,
		EventChan: eventChan,
	}
	err := s.Close(context.Background())
	assert.NotNil(t, err)
	var streamEvent interface{}
	select {
	case streamEvent = <-eventChan:
	default:
	}
	assert.Nil(t, streamEvent)
}

func Test_GetConduit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt := mocks.NewMockConduit(ctrl)
	s := &stream.Stream{
		Conduit: cndt,
	}
	assert.NotNil(t, s.GetConduit())
}

func getTargetsResponse(identifiers []string) *nspAPI.TargetResponse {
	getTargetsResponse := &nspAPI.TargetResponse{
		Targets: []*nspAPI.Target{},
	}
	for _, identifier := range identifiers {
		newTarget := &nspAPI.Target{
			Context: map[string]string{types.IdentifierKey: identifier},
		}
		getTargetsResponse.Targets = append(getTargetsResponse.Targets, newTarget)
	}
	return getTargetsResponse
}

func getConduitTrenchNSP(
	ctrl *gomock.Controller,
	trenchName string,
	conduitName string,
	ips []string) (*mocks.MockConduit, *mocks.MockTrench, *nspMock.MockTargetRegistryClient, *nspMock.MockTargetRegistry_WatchClient) {
	cndt := mocks.NewMockConduit(ctrl)
	targetRegistryClient := nspMock.NewMockTargetRegistryClient(ctrl)
	targetRegistryWatchClient := nspMock.NewMockTargetRegistry_WatchClient(ctrl)
	trnch := mocks.NewMockTrench(ctrl)
	cndt.EXPECT().GetName().Return(conduitName).AnyTimes()
	cndt.EXPECT().GetTrench().Return(trnch).AnyTimes()
	cndt.EXPECT().GetIPs().Return(ips).AnyTimes()
	trnch.EXPECT().GetName().Return(trenchName).AnyTimes()
	trnch.EXPECT().GetTargetRegistryClient().Return(targetRegistryClient).AnyTimes()
	targetRegistryClient.EXPECT().Watch(gomock.Any(), gomock.Any()).Return(targetRegistryWatchClient, nil).AnyTimes()
	return cndt, trnch, targetRegistryClient, targetRegistryWatchClient
}
