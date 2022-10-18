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
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/conduit"
	"github.com/nordix/meridio/pkg/ambassador/tap/conduit/mocks"
	typesMocks "github.com/nordix/meridio/pkg/ambassador/tap/types/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

var timeout = 500 * time.Millisecond

func Test_Manager_Run_Stop(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	manager := conduit.NewStreamManager(nil, nil, nil, nil, timeout, 30*time.Second)
	manager.Run()
	err := manager.Stop(context.TODO())
	assert.Nil(t, err)
}

func Test_RemoveStream_non_existing(t *testing.T) {
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

	manager := conduit.NewStreamManager(nil, nil, nil, nil, timeout, 30*time.Second)
	err := manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)
}

// 1. Creates the stream manager
// 2. Run the stream manager
// 3. Add a new stream
// 4. Verify Status is pending
// 5. Verify Status is open
// 6. Stop the stream manager
// 7. Verify Status is unavailable
// Check the stream has been received by the registry with the correct status
// Check Open (Stream) has been called
// Check Close (Stream) has been called
func Test_Manager_Running_AddStream_Stop(t *testing.T) {
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
	streams := []*nspAPI.Stream{
		{
			Name: "stream-a",
			Conduit: &nspAPI.Conduit{
				Name: "conduit-a",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	// 4. Verify Status is pending
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_PENDING, status)
	})
	// 5. Verify Status is open
	setStatus2 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_OPEN, status)
	}).After(setStatus1)
	// 7. Verify Status is unavailable
	r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNDEFINED, status)
	}).After(setStatus2)

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// Check Open (Stream) has been called
	openCtx, openCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	open := streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		openCancel()
		return nil
	})
	// Check Close (Stream) has been called
	streamA.EXPECT().Close(gomock.Any()).Return(nil).After(open)

	// 1. Creates the stream manager
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, timeout, 30*time.Second)
	manager.SetStreams(streams)

	// 2. Run the stream manager
	manager.Run()

	// 3. Add a new stream
	err := manager.AddStream(s)
	assert.Nil(t, err)

	// Check Open (Stream) has been called
	<-openCtx.Done()

	// 6. Stop the stream manager
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}

// 1. Creates the stream manager
// 2. Run the stream manager
// 3. Add a new stream
// 4. Verify Status is pending
// 5. Verify Status is open
// 6. Remove the stream
// 7. Verify stream has been removed
// 8. Stop the stream manager
// Check the stream has been received by the registry with the correct status
// Check Open (Stream) has been called
// Check Close (Stream) has been called
func Test_Manager_RemoveStream(t *testing.T) {
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
	streams := []*nspAPI.Stream{
		{
			Name: "stream-a",
			Conduit: &nspAPI.Conduit{
				Name: "conduit-a",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	// 4. Verify Status is pending
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_PENDING, status)
	})
	// 5. Verify Status is open
	r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_OPEN, status)
	}).After(setStatus1)
	// 7. Verify stream has been removed
	r.EXPECT().Remove(gomock.Any()).Return()

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// Check Open (Stream) has been called
	openCtx, openCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		openCancel()
		return nil
	})
	// Check Close (Stream) has been called
	streamA.EXPECT().Close(gomock.Any()).Return(nil)

	// 1. Creates the stream manager
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, timeout, 30*time.Second)
	manager.SetStreams(streams)

	// 2. Run the stream manager
	manager.Run()

	// 3. Add a new stream
	err := manager.AddStream(s)
	assert.Nil(t, err)

	// Check Open (Stream) has been called
	<-openCtx.Done()

	// 6. Remove the stream
	err = manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)

	// 8. Stop the stream manager
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}

func Test_Manager_Close_While_Opening(t *testing.T) {
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
	streams := []*nspAPI.Stream{
		{
			Name: "stream-a",
			Conduit: &nspAPI.Conduit{
				Name: "conduit-a",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_PENDING, status)
	})
	r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNAVAILABLE, status)
	}).After(setStatus1).AnyTimes()
	r.EXPECT().Remove(gomock.Any()).Return()

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// Check Open (Stream) has been called
	openCtx, openCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		openCancel()
		<-ctx.Done()
		return ctx.Err()
	}).AnyTimes()
	// Check Close (Stream) has been called
	streamA.EXPECT().Close(gomock.Any()).Return(nil)

	// 1. Creates the stream manager
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, 0, 30*time.Second)
	manager.SetStreams(streams)

	// 2. Run the stream manager
	manager.Run()

	// 3. Add a new stream
	err := manager.AddStream(s)
	assert.Nil(t, err)

	<-openCtx.Done()

	// 4. Remove the stream
	err = manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)

	// 5. Stop the stream manager
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}

// 1. Creates the stream manager
// 2. Run the stream manager
// 3. Add a new stream
// 4. Verify Status is pending
// 5. verify open is called and return err
// 6. Verify Status is unavailable
// 7. verify open is called again and return no error
// 8. Verify Status is open
// 9. Remove the stream
// 10. Stop the stream manager
func Test_Manager_Retry_Open(t *testing.T) {
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
	streams := []*nspAPI.Stream{
		{
			Name: "stream-a",
			Conduit: &nspAPI.Conduit{
				Name: "conduit-a",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	// 4. Verify Status is pending
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_PENDING, status)
	})
	// 6. Verify Status is unavailable
	setStatus2 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNAVAILABLE, status)
	}).After(setStatus1)
	// 8. Verify Status is open
	setStatus3 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_OPEN, status)
	}).After(setStatus2)
	r.EXPECT().Remove(gomock.Any()).Return().After(setStatus3)

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// 5. verify open is called and return err
	firstOpen := streamA.EXPECT().Open(gomock.Any(), gomock.Any()).Return(errors.New(""))
	// 7. verify open is called again and return no error
	secondOpenCtx, secondOpenCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	secondOpen := streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		secondOpenCancel()
		return nil
	}).After(firstOpen)
	streamA.EXPECT().Close(gomock.Any()).Return(nil).After(secondOpen)

	// 1. Creates the stream manager
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, timeout, 30*time.Second)
	manager.SetStreams(streams)

	// 2. Run the stream manager
	manager.Run()

	// 3. Add a new stream
	err := manager.AddStream(s)
	assert.Nil(t, err)

	<-secondOpenCtx.Done()

	// 9. Remove the stream
	err = manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)

	// 10. Stop the stream manager
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}

// 1. Creates the stream manager
// 2. Run the stream manager (with no NSP Stream)
// 3. Add a new stream
// 4. Verify Status is undefined
// 5. Set the NSP Streams
// 6. verify open is called
// 7. Verify Status is open
// 8. Remove the NSP Streams (set to empty list)
// 9. verify close is called
// 10. Verify Status is undefined
// 11. Remove the stream
// 12. Stop the stream manager
func Test_Manager_Add_Non_Existing_Stream(t *testing.T) {
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
	streams := []*nspAPI.Stream{
		{
			Name: "stream-a",
			Conduit: &nspAPI.Conduit{
				Name: "conduit-a",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	// 4. Verify Status is undefined
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNDEFINED, status)
	})
	// 7. Verify Status is open
	setStatus2 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_OPEN, status)
	}).After(setStatus1)
	// 10. Verify Status is undefined
	setStatus3 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNDEFINED, status)
	}).After(setStatus2)
	r.EXPECT().Remove(gomock.Any()).Return().After(setStatus3)

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// 6. verify open is called
	openCtx, openCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		openCancel()
		return nil
	})
	// 9. verify close is called
	closeCtx, closeCancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	streamA.EXPECT().Close(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		closeCancel()
		return nil
	}).AnyTimes()

	// 1. Creates the stream manager
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, timeout, 30*time.Second)

	// 2. Run the stream manager (with no NSP Streams)
	manager.Run()

	// 3. Add a new stream
	err := manager.AddStream(s)
	assert.Nil(t, err)

	// 5. Set the NSP Streams
	manager.SetStreams(streams)

	// 6. verify open is called
	<-openCtx.Done()

	// 8. Remove the NSP Streams (set to empty list)
	manager.SetStreams([]*nspAPI.Stream{})

	// 9. verify close is called
	<-closeCtx.Done()

	// 11. Remove the stream
	err = manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)

	// 12. Stop the stream manager
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}
