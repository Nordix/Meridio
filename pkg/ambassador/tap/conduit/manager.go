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
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/retry"
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
	NSPEntryTimeout            time.Duration
	// RetryDelay corresponds to the time between each Open call attempt
	RetryDelay time.Duration
	// list of streams available in the conduit.
	ConduitStreams map[string]*nspAPI.Stream
	mu             sync.Mutex
	status         status
}

func NewStreamManager(configurationManagerClient nspAPI.ConfigurationManagerClient,
	targetRegistryClient nspAPI.TargetRegistryClient,
	streamRegistry types.Registry,
	streamFactory StreamFactory,
	timeout time.Duration,
	nspEntryTimeout time.Duration) *streamManager {
	sm := &streamManager{
		Streams:                    map[string]*streamRetry{},
		ConfigurationManagerClient: configurationManagerClient,
		TargetRegistryClient:       targetRegistryClient,
		StreamRegistry:             streamRegistry,
		StreamFactory:              streamFactory,
		Timeout:                    timeout,
		NSPEntryTimeout:            nspEntryTimeout,
		ConduitStreams:             map[string]*nspAPI.Stream{},
		RetryDelay:                 1 * time.Second,
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
		Stream:          s,
		StreamRegistry:  sm.StreamRegistry,
		Timeout:         sm.Timeout,
		NSPEntryTimeout: sm.NSPEntryTimeout,
		RetryDelay:      sm.RetryDelay,
	}
	sm.Streams[strm.FullName()] = sr
	if sm.status == stopped {
		// Add to stream registry with unavailable status since the manager is not running
		sr.setStatus(ambassadorAPI.StreamStatus_UNAVAILABLE)
		return nil
	}
	nspStream := sm.getNSPStream(sr)
	if nspStream == nil {
		sr.setStatus(ambassadorAPI.StreamStatus_UNDEFINED)
		return nil
	}
	sr.setStatus(ambassadorAPI.StreamStatus_PENDING)
	sr.Open(nspStream)
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
		nspStream := sm.getNSPStream(sr)
		if nspStream != nil {
			sr.Open(nspStream)
		} else {
			ctx, cancel := context.WithTimeout(context.TODO(), sr.Timeout)
			defer cancel()
			err := sr.Close(ctx)
			sr.setStatus(ambassadorAPI.StreamStatus_UNDEFINED)
			if err != nil {
				log.Logger.Error(err, "closing non available stream", "stream", sr.Stream.GetStream())
			}
		}
	}
}

func (sm *streamManager) getNSPStream(sr *streamRetry) *nspAPI.Stream {
	return sm.ConduitStreams[sr.Stream.GetStream().FullName()]
}

// Run open all streams registered and set their
// status based on the ones available in the conduit.
func (sm *streamManager) Run() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// open all streams
	for _, sr := range sm.Streams {
		nspStream := sm.getNSPStream(sr)
		if nspStream != nil {
			sr.Open(nspStream)
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
	Stream          types.Stream
	StreamRegistry  types.Registry
	Timeout         time.Duration
	NSPEntryTimeout time.Duration
	RetryDelay      time.Duration
	currentStatus   ambassadorAPI.StreamStatus_Status
	ctxMu           sync.Mutex
	cancelOpen      context.CancelFunc
}

// Open continually tries to open the stream. The function
// finishes when the stream is successfully opened or when the
// close function is called.
func (sr *streamRetry) Open(nspStream *nspAPI.Stream) {
	// Set cancelOpen, so the close function could cancel this Open function.
	sr.ctxMu.Lock()
	if sr.cancelOpen != nil { // a previous open is still running
		sr.ctxMu.Unlock()
		return
	}
	var ctx context.Context
	ctx, sr.cancelOpen = context.WithCancel(context.TODO())
	sr.ctxMu.Unlock()
	go func() {
		// retry to refresh
		_ = retry.Do(func() error {
			openCtx, cancel := context.WithTimeout(ctx, sr.Timeout)
			defer cancel()

			// retry to open
			_ = retry.Do(func() error {
				err := sr.Stream.Open(openCtx, nspStream)
				if err != nil {
					log.Logger.Error(err, "opening stream", "stream", sr.Stream.GetStream())
					// opened unsuccessfully, set status to UNDEFINED (might be due to lack of identifier, no connection to NSP)
					sr.setStatus(ambassadorAPI.StreamStatus_UNAVAILABLE)
					return err
				}
				sr.setStatus(ambassadorAPI.StreamStatus_OPEN)
				return nil
			}, retry.WithContext(openCtx),
				retry.WithDelay(sr.RetryDelay))

			return nil
		}, retry.WithContext(ctx),
			retry.WithDelay(sr.NSPEntryTimeout),
			retry.WithErrorIngnored())
	}()
}

// Close cancel the opening of the stream and tries 1 time
// to close the stream.
// The status status has to be set by the caller.
func (sr *streamRetry) Close(ctx context.Context) error {
	sr.ctxMu.Lock()
	if sr.cancelOpen != nil {
		sr.cancelOpen() // cancel open
	}
	defer func() {
		sr.cancelOpen = nil
		sr.ctxMu.Unlock()
	}()
	return sr.Stream.Close(ctx)
}

func (sr *streamRetry) setStatus(status ambassadorAPI.StreamStatus_Status) {
	sr.currentStatus = status
	if sr.StreamRegistry == nil {
		return
	}
	sr.StreamRegistry.SetStatus(sr.Stream.GetStream(), status)
}

func convert(streams []*nspAPI.Stream) map[string]*nspAPI.Stream {
	strms := map[string]*nspAPI.Stream{}
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
		strms[ambassadorStream.FullName()] = stream
	}
	return strms
}
