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

package stream_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream/mocks"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func Test_Open_Close(t *testing.T) {
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
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	maxNumberOfTargets := 2
	identifierSelected := "0"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt := mocks.NewMockConduit(ctrl)
	tr := mocks.NewMockTargetRegistry(ctrl)
	cndt.EXPECT().GetIPs().Return(ips).AnyTimes()
	tr.EXPECT().GetTargets(gomock.Any(), gomock.Any()).Return(getTargets([]string{"1"}), nil).AnyTimes()
	firstRegister := tr.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) error {
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
		return nil
	})
	tr.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) error {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Status, nspAPI.Target_ENABLED)
		identifier, exists := target.Context[types.IdentifierKey]
		assert.True(t, exists)
		assert.Equal(t, identifier, identifierSelected)
		return nil
	}).After(firstRegister)
	tr.EXPECT().Unregister(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) error {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Stream, s.ToNSP())
		return nil
	})

	strm, err := stream.New(s, nil, maxNumberOfTargets, cndt)
	assert.Nil(t, err)
	assert.NotNil(t, strm)
	strm.TargetRegistry = tr

	err = strm.Open(context.TODO())
	assert.Nil(t, err)

	err = strm.Close(context.TODO())
	assert.Nil(t, err)
}

func Test_Open_NoIdentifierAvailable(t *testing.T) {
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
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	maxNumberOfTargets := 2

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt := mocks.NewMockConduit(ctrl)
	tr := mocks.NewMockTargetRegistry(ctrl)
	cndt.EXPECT().GetIPs().Return(ips).AnyTimes()
	tr.EXPECT().GetTargets(gomock.Any(), gomock.Any()).Return(getTargets([]string{"1", "2"}), nil).AnyTimes()

	strm, err := stream.New(s, nil, maxNumberOfTargets, cndt)
	assert.Nil(t, err)
	assert.NotNil(t, strm)
	strm.TargetRegistry = tr

	err = strm.Open(context.TODO())
	assert.NotNil(t, err)
}

func Test_Open_Concurrent(t *testing.T) {
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
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	maxNumberOfTargets := 3
	identifierSelected := "0"
	concurrentIdentifier := "0"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt := mocks.NewMockConduit(ctrl)
	tr := mocks.NewMockTargetRegistry(ctrl)
	cndt.EXPECT().GetIPs().Return(ips).AnyTimes()
	firstGet := tr.EXPECT().GetTargets(gomock.Any(), gomock.Any()).Return(getTargets([]string{"1"}), nil)
	secondGet := tr.EXPECT().GetTargets(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) ([]*nspAPI.Target, error) {
		concurrentIdentifier = identifierSelected
		return getTargets([]string{"1", concurrentIdentifier, identifierSelected}), nil
	}).After(firstGet)
	tr.EXPECT().GetTargets(gomock.Any(), gomock.Any()).Return(getTargets([]string{"1", concurrentIdentifier, identifierSelected}), nil).After(secondGet)

	firstRegister := tr.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) error {
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
		return nil
	})
	secondRegister := tr.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) error {
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
		return nil
	}).After(firstRegister)
	tr.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) error {
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		assert.Equal(t, target.Status, nspAPI.Target_ENABLED)
		identifier, exists := target.Context[types.IdentifierKey]
		assert.True(t, exists)
		assert.Equal(t, identifier, identifierSelected)
		return nil
	}).After(secondRegister)

	strm, err := stream.New(s, nil, maxNumberOfTargets, cndt)
	assert.Nil(t, err)
	assert.NotNil(t, strm)
	strm.TargetRegistry = tr

	err = strm.Open(context.TODO())
	assert.Nil(t, err)
}

func Test_Open_Concurrent_NoIdentifierAvailable(t *testing.T) {
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
	ips := []string{"172.16.0.1/24", "fd00::1/64"}
	maxNumberOfTargets := 2
	identifierSelected := "0"
	concurrentIdentifier := "0"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cndt := mocks.NewMockConduit(ctrl)
	tr := mocks.NewMockTargetRegistry(ctrl)
	cndt.EXPECT().GetIPs().Return(ips).AnyTimes()
	firstGet := tr.EXPECT().GetTargets(gomock.Any(), gomock.Any()).Return(getTargets([]string{"1"}), nil)
	tr.EXPECT().GetTargets(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) ([]*nspAPI.Target, error) {
		concurrentIdentifier = identifierSelected
		return getTargets([]string{"1", concurrentIdentifier, identifierSelected}), nil
	}).After(firstGet)

	tr.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) error {
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
		return nil
	})
	tr.EXPECT().Unregister(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, target *nspAPI.Target) error {
		assert.NotNil(t, target)
		assert.NotNil(t, target)
		assert.Equal(t, target.Ips, ips)
		return nil
	})

	strm, err := stream.New(s, nil, maxNumberOfTargets, cndt)
	assert.Nil(t, err)
	assert.NotNil(t, strm)
	strm.TargetRegistry = tr

	err = strm.Open(context.TODO())
	assert.NotNil(t, err)
}

func getTargets(identifiers []string) []*nspAPI.Target {
	targets := []*nspAPI.Target{}
	for _, identifier := range identifiers {
		newTarget := &nspAPI.Target{
			Context: map[string]string{types.IdentifierKey: identifier},
		}
		targets = append(targets, newTarget)
	}
	return targets
}
