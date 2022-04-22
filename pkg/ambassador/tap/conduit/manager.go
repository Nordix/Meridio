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

package conduit

import (
	"context"
	"fmt"
	"sync"
	"time"

	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/sirupsen/logrus"
)

const (
	stopped = iota
	running
)

type status int

// streamManager is responsible for:
// - opening/closing streams based of the streams available in the conduit.
// - Re-opening streams which have been closed by another resource (NSP failures...).
// - setting the status of the streams
type streamManager struct {
	// contains the streams managed by this stream manager
	Streams                    map[string]*streamRetry
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	TargetRegistryClient       nspAPI.TargetRegistryClient
	StreamRegistry             types.Registry
	StreamFactory              StreamFactory
	Timeout                    time.Duration
	// list of streams available in the conduit.
	ConduitStreams map[string]*ambassadorAPI.Stream
	mu             sync.Mutex
	status         status
}

func NewStreamManager(configurationManagerClient nspAPI.ConfigurationManagerClient,
	targetRegistryClient nspAPI.TargetRegistryClient,
	streamRegistry types.Registry,
	streamFactory StreamFactory,
	timeout time.Duration) StreamManager {
	sm := &streamManager{
		Streams:                    map[string]*streamRetry{},
		ConfigurationManagerClient: configurationManagerClient,
		TargetRegistryClient:       targetRegistryClient,
		StreamRegistry:             streamRegistry,
		StreamFactory:              streamFactory,
		Timeout:                    timeout,
		ConduitStreams:             map[string]*ambassadorAPI.Stream{},
		status:                     stopped,
	}
	return sm
}

// AddStream adds the stream to the stream manager, registers it to the
// stream registry, creates a new stream based on StreamFactory, and open it, if
// the stream manager is running and if the stream exists in the configuration.
func (sm *streamManager) AddStream(strm *ambassadorAPI.Stream) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	_, exists := sm.Streams[strm.FullName()]
	if exists {
		return nil
	}
	s, err := sm.StreamFactory.New(strm)
	if err != nil {
		return err
	}
	sr := &streamRetry{
		Stream:         s,
		StreamRegistry: sm.StreamRegistry,
		Timeout:        sm.Timeout,
	}
	sm.Streams[strm.FullName()] = sr
	if sm.status == stopped {
		// Add to stream registry with unavailable status since the manager is not running
		sr.setStatus(ambassadorAPI.StreamStatus_UNAVAILABLE)
		return nil
	}
	if !sm.streamExists(sr) {
		sr.setStatus(ambassadorAPI.StreamStatus_UNDEFINED)
		return nil
	}
	sr.setStatus(ambassadorAPI.StreamStatus_PENDING)
	go sr.Open()
	return nil
}

// RemoveStream removes the stream from the manager, removes it
// from the stream registry and closes it.
// TODO: Error handling, if failed to close
func (sm *streamManager) RemoveStream(ctx context.Context, strm *ambassadorAPI.Stream) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// check if stream exists
	stream, exists := sm.Streams[strm.FullName()]
	if !exists {
		return nil
	}
	// Close the stream
	err := stream.Close(ctx) // todo: retry
	// remove the stream for the stream registry (the watchers will get notified
	// from this change).
	sm.StreamRegistry.Remove(strm)
	// delete it from the stream manager
	delete(sm.Streams, strm.FullName())
	return err
}

// GetStreams returns the list of streams (opened or not).
func (sm *streamManager) GetStreams() []*ambassadorAPI.Stream {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Get all streams in a new list
	streams := []*ambassadorAPI.Stream{}
	for _, sr := range sm.Streams {
		streams = append(streams, sr.Stream.GetStream())
	}
	return streams
}

// Set all streams available in the conduit
func (sm *streamManager) SetStreams(streams []*nspAPI.Stream) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.ConduitStreams = convert(streams)
	if sm.status == stopped {
		return
	}
	// check streams to open
	for _, sr := range sm.Streams {
		if sm.streamExists(sr) {
			go sr.Open()
		} else {
			ctx, cancel := context.WithTimeout(context.TODO(), sr.Timeout)
			defer cancel()
			err := sr.Close(ctx)
			sr.setStatus(ambassadorAPI.StreamStatus_UNDEFINED)
			if err != nil {
				logrus.Errorf("error closing non available stream (%v): %v", sr.Stream.GetStream(), err)
			}
		}
	}
}

func (sm *streamManager) streamExists(sr *streamRetry) bool {
	_, exists := sm.ConduitStreams[sr.Stream.GetStream().FullName()]
	return exists
}

// Run open all streams registered and set their
// status based on the ones available in the conduit.
func (sm *streamManager) Run() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// open all streams
	for _, sr := range sm.Streams {
		if sm.streamExists(sr) {
			go sr.Open()
		} else {
			sr.setStatus(ambassadorAPI.StreamStatus_UNDEFINED)
		}
	}
	sm.status = running
}

// Stop closes all streams
func (sm *streamManager) Stop(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// WaitGroup to wait for all stream close to be done
	var wg sync.WaitGroup
	wg.Add(len(sm.Streams))
	// final error concatenating all errors
	var errFinal error
	// mutex to avoid concurrency write on errFinal
	var mu sync.Mutex
	for _, stream := range sm.Streams {
		go func(s *streamRetry) {
			defer wg.Done()
			err := s.Close(ctx)
			s.setStatus(ambassadorAPI.StreamStatus_UNDEFINED)
			if err != nil {
				mu.Lock()
				errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
				mu.Unlock()
			}
		}(stream)
	}
	wg.Wait()
	sm.status = stopped
	return errFinal
}

type streamRetry struct {
	Stream         types.Stream
	StreamRegistry types.Registry
	Timeout        time.Duration
	currentStatus  ambassadorAPI.StreamStatus_Status
	mu             sync.Mutex
	ctxMu          sync.Mutex
	statusMu       sync.Mutex
	cancelOpen     context.CancelFunc
}

// Open continually tries to open the stream. The function
// finishes when the stream is successfully opened or when the
// close function is called.
// This function is excepted to run as a goroutine.
func (sr *streamRetry) Open() {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if sr.getStatus() == ambassadorAPI.StreamStatus_OPEN {
		return
	}

	// Set cancelOpen, so the close function could cancel this Open function.
	sr.ctxMu.Lock()
	var ctx context.Context
	ctx, sr.cancelOpen = context.WithCancel(context.TODO())
	sr.ctxMu.Unlock()

	// retry to refresh
	_ = retry.Do(func() error {
		openCtx, cancel := context.WithTimeout(ctx, sr.Timeout)
		defer cancel()

		// retry to open
		_ = retry.Do(func() error {
			err := sr.Stream.Open(openCtx)
			if err != nil {
				logrus.Warnf("error opening stream: %v ; %v", sr.Stream.GetStream(), err)
				// opened unsuccessfully, set status to UNDEFINED (might be due to lack of identifier, no connection to NSP)
				sr.setStatus(ambassadorAPI.StreamStatus_UNAVAILABLE)
				return err
			}
			sr.setStatus(ambassadorAPI.StreamStatus_OPEN)
			return nil
		}, retry.WithContext(openCtx),
			retry.WithDelay(50*time.Millisecond))

		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(30*time.Second),
		retry.WithErrorIngnored())
}

// Close cancel the opening of the stream and tries 1 time
// to close the stream.
// The status status has to be set by the caller.
func (sr *streamRetry) Close(ctx context.Context) error {
	sr.ctxMu.Lock()
	if sr.cancelOpen != nil {
		sr.cancelOpen() // cancel open
	}
	sr.ctxMu.Unlock()
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return sr.Stream.Close(ctx)
}

func (sr *streamRetry) setStatus(status ambassadorAPI.StreamStatus_Status) {
	sr.statusMu.Lock()
	defer sr.statusMu.Unlock()
	sr.currentStatus = status
	if sr.StreamRegistry == nil {
		return
	}
	sr.StreamRegistry.SetStatus(sr.Stream.GetStream(), status)
}

func (sr *streamRetry) getStatus() ambassadorAPI.StreamStatus_Status {
	sr.statusMu.Lock()
	defer sr.statusMu.Unlock()
	return sr.currentStatus
}

func convert(streams []*nspAPI.Stream) map[string]*ambassadorAPI.Stream {
	strms := map[string]*ambassadorAPI.Stream{}
	for _, stream := range streams {
		if stream == nil || stream.GetConduit() == nil || stream.GetConduit().GetTrench() == nil {
			continue
		}
		ambassadorStream := &ambassadorAPI.Stream{
			Name: stream.GetName(),
			Conduit: &ambassadorAPI.Conduit{
				Name: stream.GetConduit().Name,
				Trench: &ambassadorAPI.Trench{
					Name: stream.GetConduit().GetTrench().GetName(),
				},
			},
		}
		strms[ambassadorStream.FullName()] = ambassadorStream
	}
	return strms
}
