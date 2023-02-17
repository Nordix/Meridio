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

package trench

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/trench/mocks"
	typesMocks "github.com/nordix/meridio/pkg/ambassador/tap/types/mocks"
	"github.com/nordix/meridio/test/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func Test_Equals(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	c := &ambassadorAPI.Conduit{
		Name: "conduit-a",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	trnch := c.GetTrench()
	trnchB := &ambassadorAPI.Trench{
		Name: "trench-b",
	}

	trench := Trench{
		Trench: trnch,
		logger: logr.Discard(),
	}
	assert.True(t, trench.Equals(trnch))
	assert.False(t, trench.Equals(trnchB))
}

func Test_AddConduit_RemoveConduit(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	c := &ambassadorAPI.Conduit{
		Name: "conduit-a",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	trnch := c.GetTrench()

	var wg sync.WaitGroup

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	conduitFactory := mocks.NewMockConduitFactory(ctrl)
	conduitA := typesMocks.NewMockConduit(ctrl)
	conduitFactory.EXPECT().New(gomock.Any()).Return(conduitA, nil)
	conduitA.EXPECT().GetConduit().Return(c)
	conduitA.EXPECT().Equals(gomock.Any()).Return(true)

	conduitA.EXPECT().Connect(gomock.Any()).DoAndReturn(func(_ context.Context) error {
		defer wg.Done()
		return nil
	})
	conduitA.EXPECT().Disconnect(gomock.Any()).Return(nil)

	trench := Trench{
		Trench:         trnch,
		ConduitFactory: conduitFactory,
		logger:         logr.Discard(),
	}

	wg.Add(1)
	conduit, err := trench.AddConduit(context.TODO(), c)
	assert.Nil(t, err)
	assert.Equal(t, conduitA, conduit)
	conduits := trench.GetConduits()
	assert.Len(t, conduits, 1)
	assert.Contains(t, conduits, conduitA)

	err = utils.WaitTimeout(&wg, utils.TestTimeout) // wait for Connect call
	assert.Nil(t, err)

	err = trench.RemoveConduit(context.TODO(), c)
	assert.Nil(t, err)
}

func Test_AddConduit_RemoveConduit_WhileConnecting(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	c := &ambassadorAPI.Conduit{
		Name: "conduit-a",
		Trench: &ambassadorAPI.Trench{
			Name: "trench-a",
		},
	}
	trnch := c.GetTrench()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	conduitFactory := mocks.NewMockConduitFactory(ctrl)
	conduitA := typesMocks.NewMockConduit(ctrl)
	conduitFactory.EXPECT().New(gomock.Any()).Return(conduitA, nil)
	conduitA.EXPECT().Equals(gomock.Any()).Return(true)
	conduitA.EXPECT().GetConduit().Return(c).AnyTimes()

	connectCtx, connectCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	firstConnect := conduitA.EXPECT().Connect(gomock.Any()).DoAndReturn(func(_ context.Context) error {
		connectCancel()
		return errors.New("")
	})
	conduitA.EXPECT().Connect(gomock.Any()).Return(errors.New("")).After(firstConnect).AnyTimes()
	conduitA.EXPECT().Disconnect(gomock.Any()).Return(nil)

	trench := Trench{
		Trench:         trnch,
		ConduitFactory: conduitFactory,
		logger:         logr.Discard(),
	}

	conduit, err := trench.AddConduit(context.TODO(), c)
	assert.Nil(t, err)
	assert.Equal(t, conduitA, conduit)
	conduits := trench.GetConduits()
	assert.Len(t, conduits, 1)
	assert.Contains(t, conduits, conduitA)

	<-connectCtx.Done()

	err = trench.RemoveConduit(context.TODO(), c)
	assert.Nil(t, err)
}
