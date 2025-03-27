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

package conduit

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/go-logr/logr"
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
	logger         logr.Logger
	conduitIsDown  bool // conduit connectivity down, currently does not affect NSP registration
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
		logger:                     log.Logger.WithValues("class", "streamManager"),
	}
	return sm
}

// AddStream adds the stream to the stream manager, registers it to the
// stream registry, creates a new stream based on StreamFactory, and open it, if
// the stream manager is running and if the stream exists in the configuration (NSP).
func (sm *streamManager) AddStream(strm *ambassadorAPI.Stream) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	_, exists := sm.Streams[strm.FullName()]
	if exists {
		return nil
	}
	s, err := sm.StreamFactory.New(strm)
	if err != nil {
		return fmt.Errorf("stream factory create failed: %w", err)
	}
	// TODO: streamRetry with initial ambassadorAPI status OPEN seems weird
	sr := &streamRetry{
		Stream:          s,
		StreamRegistry:  sm.StreamRegistry,
		Timeout:         sm.Timeout,
		NSPEntryTimeout: sm.NSPEntryTimeout,
		RetryDelay:      sm.RetryDelay,
		conduitIsDown:   sm.conduitIsDown,
	}
	sm.Streams[strm.FullName()] = sr
	if sm.status == stopped {
		// Add to stream registry with unavailable status since the manager is not running
		sr.setStatus(ambassadorAPI.StreamStatus_UNAVAILABLE)
		return nil
	}
	nspStream := sm.getNSPStream(sr)
	if nspStream == nil {
		// Not in configuration (NSP), wait for SetStreams()
		sr.setStatus(ambassadorAPI.StreamStatus_UNDEFINED)
		return nil
	}
	sr.setStatus(ambassadorAPI.StreamStatus_PENDING)
	sr.Open(nspStream, false)
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
			sr.Open(nspStream, false)
		} else {
			ctx, cancel := context.WithTimeout(context.TODO(), sr.Timeout)
			defer cancel()
			err := sr.Close(ctx)
			sr.setStatus(ambassadorAPI.StreamStatus_UNDEFINED)
			if err != nil {
				sm.logger.Error(err, "closing non available stream", "stream", sr.Stream.GetStream())
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
			sr.Open(nspStream, false)
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
				errFinal = fmt.Errorf("%w; stream manager stream close failure: %w", errFinal, err) // todo
				mu.Unlock()
			}
		}(stream)
	}
	wg.Wait()
	sm.status = stopped
	return errFinal
}

func (sm *streamManager) ConduitDown(isDown bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.conduitIsDown == isDown {
		return
	}

	sm.conduitIsDown = isDown
	var wg sync.WaitGroup
	wg.Add(len(sm.Streams))
	for _, stream := range sm.Streams {
		go func(s *streamRetry) {
			defer wg.Done()
			nspStream := sm.getNSPStream(stream)
			stream.conduitChange(isDown, nspStream)
		}(stream)
	}
	wg.Wait()
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
	conduitIsDown   bool // conduit connectivity down, indicate ambassador API users if stream can carry traffic
	statusMu        sync.Mutex
}

// Open continuously tries to open the stream. When opened it continuously
// tries to refresh it to keep it open.
// Calling cancelOpen() stops the open/refresh goroutine.
//
// Note: Force=true cancels previous open/refresh goroutine and starts new
// one with a short random delay. A canceled open/refresh goroutine will
// update the stream status to UNAVAILABLE. Hence, avoid using force=true
// unless conduitIsDown=true to avoid misleading status changes. Or make
// sure to uniformly set stream status in caller (i.e. streamManager) when
// cancelling a streamRetry open.
func (sr *streamRetry) Open(nspStream *nspAPI.Stream, force bool) {
	// Set cancelOpen, so the close function could cancel this Open function.
	sr.ctxMu.Lock()
	if sr.cancelOpen != nil { // a previous open is still running
		if !force {
			sr.ctxMu.Unlock()
			return
		}
		log.Logger.V(1).Info("Forced stream re-open to ensure stream status is up to date", "stream", sr.Stream.GetStream())
		sr.cancelOpen() // force cancels former open goroutine to trigger a quick re-open
	}
	var ctx context.Context
	ctx, sr.cancelOpen = context.WithCancel(context.TODO())
	sr.ctxMu.Unlock()
	go func() {
		if force {
			// Add a short random delay in an attempt to avoid burst of open calls
			// in the case of multipe opened Streams per Conduit. (Also, lowers risk
			// of race with a previous lingering open/refresh goroutine to occur in
			// the case of force open.)
			randomDelay := 10*time.Millisecond + time.Duration(rand.Intn(90))*time.Millisecond // 10-100ms delay
			log.Logger.V(1).Info("delay initial open", "delay ms", randomDelay.Milliseconds(), "stream", sr.Stream.GetStream())
			timer := time.NewTimer(randomDelay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				log.Logger.V(1).Info("open context cancelled during initial delay", "stream", sr.Stream.GetStream())
				timer.Stop()
				select { // Drain timer channel if it fired concurrently with ctx cancel
				case <-timer.C:
				default:
				}
				return // Context cancelled, exit goroutine
			}
		}
		// retry to refresh
		_ = retry.Do(func() error {
			// retry to open
			_ = retry.Do(func() error {
				openCtx, cancel := context.WithTimeout(ctx, sr.Timeout)
				defer cancel()
				// WARNING: Stream.Open calls conduit's GetIPs(), which could lead to
				// deadlock because of the conduit mutex in case a cancelOpen action
				// via the conduit would wait for the old open/refresh goroutine to
				// terminate before returning. So, rather risk a race between an old
				// and a new goroutine when it comes to setStatus() calls.
				err := sr.Stream.Open(openCtx, nspStream)
				if err != nil {
					if ctx.Err() != context.Canceled { // if cancelOpen got called by Close() no need to complain
						log.Logger.Error(err, "error opening stream", "stream", nspStream)
						// TODO: check if it really makes sense to keep retrying with high frequency
						// in case of error wraps "no identifier available to register the target"?
					}
					// opened unsuccessfully, set status to UNAVAILABLE (might be due to lack of identifier, no connection to NSP)
					sr.setStatus(ambassadorAPI.StreamStatus_UNAVAILABLE)
					return fmt.Errorf("failed to open stream: %w", err) // (make wrapcheck happy)
				}
				sr.setStatus(ambassadorAPI.StreamStatus_OPEN)
				return nil
			}, retry.WithContext(ctx),
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
	if err := sr.Stream.Close(ctx); err != nil {
		return fmt.Errorf("stream retry failed to close stream: %w", err)
	}
	return nil
}

func (sr *streamRetry) setStatus(status ambassadorAPI.StreamStatus_Status) {
	sr.statusMu.Lock()
	defer sr.statusMu.Unlock()

	if sr.conduitIsDown && status == ambassadorAPI.StreamStatus_OPEN {
		log.Logger.Info("ambassadorAPI stream status not set to OPEN due to conduit being down",
			"stream", sr.Stream.GetStream(), "current status", sr.currentStatus)
		return
	}

	if status != sr.currentStatus {
		log.Logger.Info("ambassadorAPI stream status changed", "status", status, "stream", sr.Stream.GetStream())
	}
	sr.currentStatus = status
	if sr.StreamRegistry == nil {
		return
	}
	sr.StreamRegistry.SetStatus(sr.Stream.GetStream(), status)
}

func (sr *streamRetry) conduitChange(isDown bool, nspStream *nspAPI.Stream) {
	sr.statusMu.Lock()
	// set conduitDown variable to reflect conduit connectivity status
	if sr.conduitIsDown != isDown {
		log.Logger.V(1).Info("Conduit connectivity status changed", "isDown", isDown, "stream", sr.Stream.GetStream(), "nspStream", nspStream)
		sr.conduitIsDown = isDown
	}
	sr.statusMu.Unlock()

	if isDown {
		// update status to UNAVAILABLE if conduit connectivity is down
		sr.setStatus(ambassadorAPI.StreamStatus_UNAVAILABLE)
		return
	}

	// in case the conduit connectivity is restored force stream re-open to
	// allow a quick user update (assuming open succeeds)
	// Note: In case of multiple Streams per Conduit the refresh periods could
	// get synced, which might lead to CPU spikes. Thus, Open() adds a short
	// random delay at the start of the background goroutine taking care of
	// open/refresh tasks.
	if nspStream != nil {
		sr.Open(nspStream, true)
	}
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
