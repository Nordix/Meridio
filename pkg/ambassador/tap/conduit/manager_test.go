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
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/conduit"
	"github.com/nordix/meridio/pkg/ambassador/tap/conduit/mocks"
	typesMocks "github.com/nordix/meridio/pkg/ambassador/tap/types/mocks"
	"github.com/nordix/meridio/test/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

var timeout = 500 * time.Millisecond

func Test_Manager_Run_Stop(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	manager := conduit.NewStreamManager(nil, nil, nil, nil, timeout, 30*time.Second)
	manager.RetryDelay = 1 * time.Millisecond
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
	manager.RetryDelay = 1 * time.Millisecond
	err := manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)
}

// 1. Create the stream manager with a stream set
// 2. Run the stream manager
// 3. Add (open) a new stream
// 4. Verify Status is pending
// 5. Verify Status is open
// 6. Stop the stream manager
// 7. Verify Status is undefined
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

	var wg sync.WaitGroup

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	// 4.
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_PENDING, status)
	})
	// 5.
	setStatus2 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_OPEN, status)
	}).After(setStatus1)
	// 7.
	r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNDEFINED, status)
	}).After(setStatus2)

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// Check Open (Stream) has been called
	open := streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		defer wg.Done()
		return nil
	})
	// Check Close (Stream) has been called
	streamA.EXPECT().Close(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		defer wg.Done()
		return nil
	}).After(open)

	// 1.
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, timeout, 30*time.Second)
	manager.RetryDelay = 1 * time.Millisecond
	manager.SetStreams(streams)

	// 2.
	manager.Run()

	// 3.
	wg.Add(3)
	err := manager.AddStream(s)
	assert.Nil(t, err)

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for SetStatus + Open + SetStatus

	// 6.
	wg.Add(2)
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for Close + SetStatus
}

// 1. Create the stream manager with a stream set
// 2. Run the stream manager
// 3. Add (open) a new stream
// 4. Verify Status is pending
// 5. Verify Status is open
// 6. Remove (close) the stream
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

	var wg sync.WaitGroup

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)

	// 7.
	r.EXPECT().Remove(gomock.Any()).Return()

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()

	// 4.
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_PENDING, status)
	})
	// Check Open (Stream) has been called
	open := streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		defer wg.Done()
		return nil
	}).After(setStatus1)
	// 5.
	r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_OPEN, status)
	}).After(open)

	// Check Close (Stream) has been called
	streamA.EXPECT().Close(gomock.Any()).Return(nil)

	// 1.
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, timeout, 30*time.Second)
	manager.RetryDelay = 1 * time.Millisecond
	manager.SetStreams(streams)

	// 2.
	manager.Run()

	// 3.
	wg.Add(3)
	err := manager.AddStream(s)
	assert.Nil(t, err)

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for SetStatus + Open + SetStatus

	// 6.
	err = manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)

	// 8.
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}

// 1. Create the stream manager with a stream set
// 2. Run the stream manager
// 3. Add (open) a new stream
// 4. Verify Status is pending
// 5. Remove (close) the stream
// 6. Verify Status is unavailable
// 7. Stop the stream manager
// Check the stream has been received by the registry with the correct status
// Check Open (Stream) has been called
// Check Close (Stream) has been called
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

	var wg sync.WaitGroup

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	// 4.
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_PENDING, status)
	})
	// 6.
	r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNAVAILABLE, status)
	}).After(setStatus1).AnyTimes()
	r.EXPECT().Remove(gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream) {
		defer wg.Done()
		assert.Equal(t, s, strm)
	})

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// Check Open (Stream) has been called
	streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		wg.Done()
		<-ctx.Done()
		return fmt.Errorf("streamA Open DoAndReturn: %w", ctx.Err())
	}).AnyTimes()
	// Check Close (Stream) has been called
	streamA.EXPECT().Close(gomock.Any()).Return(nil)

	// 1.
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, 0, 30*time.Second)
	manager.RetryDelay = 1 * time.Millisecond
	manager.SetStreams(streams)

	// 2.
	manager.Run()

	// 3.
	wg.Add(2)
	err := manager.AddStream(s)
	assert.Nil(t, err)

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for SetStatus + Open

	// 5.
	wg.Add(1)
	err = manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for Remove

	// 7.
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}

// 1. Create the stream manager with a stream set
// 2. Run the stream manager
// 3. Add (open) a new stream
// 4. Verify Status is pending
// 5. verify open is called and return err
// 6. Verify Status is unavailable
// 7. verify open is called again and return no error
// 8. Verify Status is open
// 9. Remove (close) the stream
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

	var wg sync.WaitGroup

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	// 4.
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_PENDING, status)
	})
	// 6.
	setStatus2 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNAVAILABLE, status)
	}).After(setStatus1)
	// 8.
	setStatus3 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_OPEN, status)
	}).After(setStatus2)
	r.EXPECT().Remove(gomock.Any()).Return().After(setStatus3)

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// 5.
	firstOpen := streamA.EXPECT().Open(gomock.Any(), gomock.Any()).Return(errors.New(""))
	// 7.
	secondOpen := streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		defer wg.Done()
		return nil
	}).After(firstOpen)
	streamA.EXPECT().Close(gomock.Any()).Return(nil).After(secondOpen)

	// 1.
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, timeout, 30*time.Second)
	manager.RetryDelay = 1 * time.Millisecond
	manager.SetStreams(streams)

	// 2.
	manager.Run()

	// 3.
	wg.Add(4)
	err := manager.AddStream(s)
	assert.Nil(t, err)

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for SetStatus + SetStatus + Open + SetStatus

	// 9.
	err = manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)

	// 10.
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}

// 1. Create the stream manager
// 2. Run the stream manager (with no NSP Stream)
// 3. Add (open) a new stream
// 4. Verify Status is undefined
// 5. Set the NSP Streams
// 6. verify open is called
// 7. Verify Status is open
// 8. Remove the NSP Streams (set to empty list)
// 9. verify close is called
// 10. Verify Status is undefined
// 11. Remove (close) the stream
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

	var wg sync.WaitGroup

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	// 4.
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNDEFINED, status)
	})
	// 7.
	setStatus2 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_OPEN, status)
	}).After(setStatus1)
	// 10.
	setStatus3 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNDEFINED, status)
	}).After(setStatus2)
	r.EXPECT().Remove(gomock.Any()).Return().After(setStatus3)

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// 6.
	streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		defer wg.Done()
		return nil
	})
	// 9.
	firstClose := streamA.EXPECT().Close(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		defer wg.Done()
		return nil
	})
	streamA.EXPECT().Close(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		return nil
	}).After(firstClose).AnyTimes()

	// 1.
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, timeout, 30*time.Second)
	manager.RetryDelay = 1 * time.Millisecond

	// 2.
	manager.Run()

	// 3.
	wg.Add(1)
	err := manager.AddStream(s)
	assert.Nil(t, err)

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for set status

	// 5.
	wg.Add(2)
	manager.SetStreams(streams)

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for open call + set status

	// 8.
	wg.Add(2)
	manager.SetStreams([]*nspAPI.Stream{})

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for close + set status

	// 11.
	err = manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)

	// 12.
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}

// 1. Create the stream manager with a stream set
// 2. Run the stream manager
// 3. Add (open) a new stream
// 4. Verify Status is pending
// 5. Verify Status is open
// 6. Trigger SetStreams many times
// 7. Remove (close) the stream
// 8. Verify stream has been removed
// 9. Stop the stream manager
// If multiple events occur (AddStream, SetStreams), a stack of Open would
// pile up waiting for the mutex to be unlocked. When the mutex would be
// unlocked (on close call), the close could then be executed, and after,
// piled open calls would execute again, which would cause the stream to be
// re-opened. This test should cover this specific case.
func Test_Manager_Add_Remove_Concurrent_Event(t *testing.T) {
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

	var wg sync.WaitGroup

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := typesMocks.NewMockRegistry(ctrl)
	// 4.
	setStatus1 := r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_PENDING, status)
	})
	// 5.
	r.EXPECT().SetStatus(gomock.Any(), gomock.Any()).DoAndReturn(func(strm *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
		defer wg.Done()
		assert.Equal(t, s, strm)
		assert.Equal(t, ambassadorAPI.StreamStatus_UNAVAILABLE, status)
	}).After(setStatus1)
	// 8.
	r.EXPECT().Remove(gomock.Any()).Return()

	streamFactory := mocks.NewMockStreamFactory(ctrl)
	streamA := typesMocks.NewMockStream(ctrl)
	streamFactory.EXPECT().New(gomock.Any()).Return(streamA, nil)
	streamA.EXPECT().GetStream().Return(s).AnyTimes()
	// Check Open (Stream) has been called
	firstOpen := streamA.EXPECT().Open(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, nspStream *nspAPI.Stream) error {
		defer wg.Done()
		return errors.New("")
	})
	// Check Close (Stream) has been called
	streamA.EXPECT().Close(gomock.Any()).Return(nil).After(firstOpen)

	// 1.
	manager := conduit.NewStreamManager(nil, nil, r, streamFactory, timeout, 30*time.Second)
	manager.RetryDelay = 5000 * time.Millisecond
	manager.SetStreams(streams)

	// 2.
	manager.Run()

	// 3.
	wg.Add(3)
	err := manager.AddStream(s)
	assert.Nil(t, err)

	// 6.
	for i := 1; i < 100; i++ { // TODO: to be fixed, how to test this?
		go manager.SetStreams(streams)
	}

	err = utils.WaitTimeout(&wg, utils.TestTimeout)
	assert.Nil(t, err) // wait for SetStatus + Open + SetStatus

	// 7.
	err = manager.RemoveStream(context.TODO(), s)
	assert.Nil(t, err)

	// 9.
	err = manager.Stop(context.TODO())
	assert.Nil(t, err)
}
