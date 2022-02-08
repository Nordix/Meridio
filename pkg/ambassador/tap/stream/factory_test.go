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
	"testing"

	"github.com/golang/mock/gomock"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func Test_New(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	logrus.SetLevel(logrus.FatalLevel)

	maxNumberOfTargets := 100
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	c := mocks.NewMockConduit(ctrl)

	sf := stream.NewFactory(nil, maxNumberOfTargets, c)
	assert.NotNil(t, sf)

	s := &ambassadorAPI.Stream{
		Name: "stream-a",
		Conduit: &ambassadorAPI.Conduit{
			Name: "conduit-a",
			Trench: &ambassadorAPI.Trench{
				Name: "trench-a",
			},
		},
	}
	typesStream, err := sf.New(s)
	assert.Nil(t, err)
	assert.NotNil(t, typesStream)

	stream, ok := typesStream.(*stream.Stream)
	assert.True(t, ok)
	assert.NotNil(t, stream)
	assert.Equal(t, maxNumberOfTargets, stream.MaxNumberOfTargets)
	assert.Equal(t, c, stream.Conduit)
}
