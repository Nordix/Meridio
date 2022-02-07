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

package stream

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	lbTypes "github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/sirupsen/logrus"
)

// Stream implements types.Stream
type Stream struct {
	Stream                     *ambassadorAPI.Stream
	TargetRegistry             TargetRegistry
	StreamRegistry             types.Registry
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	Conduit                    Conduit
	MaxNumberOfTargets         int // Maximum number of targets registered in this stream
	Configuration              Configuration
	identifier                 int
	targetStatus               nspAPI.Target_Status
	ips                        []string
	mu                         sync.Mutex
	configurationCancel        context.CancelFunc
	pendingCancel              context.CancelFunc
}

// New is the constructor of Stream.
// The constructor will add the stream to the stream registry and update its status.
// If the status is still disabled after the pendingTrigger chan has received a value,
// the status will become unavailable.
func New(stream *ambassadorAPI.Stream,
	targetRegistryClient nspAPI.TargetRegistryClient,
	configurationManagerClient nspAPI.ConfigurationManagerClient,
	streamRegistry types.Registry,
	maxNumberOfTargets int,
	pendingTrigger <-chan interface{},
	conduit Conduit) (*Stream, error) {
	// todo: check if stream valid
	s := &Stream{
		Stream:                     stream,
		TargetRegistry:             newTargetRegistryImpl(targetRegistryClient),
		ConfigurationManagerClient: configurationManagerClient,
		StreamRegistry:             streamRegistry,
		Conduit:                    conduit,
		MaxNumberOfTargets:         maxNumberOfTargets,
		identifier:                 -1,
		targetStatus:               nspAPI.Target_DISABLED,
	}
	err := s.StreamRegistry.Add(context.TODO(), s.Stream, ambassadorAPI.StreamStatus_PENDING)
	if err != nil {
		return nil, err
	}
	var cancelPendingCtx context.Context
	cancelPendingCtx, s.pendingCancel = context.WithCancel(context.TODO())
	s.setPendingStatus(pendingTrigger, cancelPendingCtx)
	s.Configuration = newConfigurationImpl(s, s.Stream.ToNSP(), s.ConfigurationManagerClient)
	return s, nil
}

// Open the stream in the conduit by generating a identifier and registering
// the target to the NSP service while avoiding the identifier collisions.
// If success, no error will be returned, and a gorouting will monitor the availability
// of the stream in the trench and update the status of the stream accordingly in
// the stream regsitry.
// If not, an error will be returned.
func (s *Stream) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ips = s.Conduit.GetIPs()
	if len(s.ips) <= 0 {
		return errors.New("Ips not set")
	}
	logrus.Infof("Attempt to open stream: %v", s.Stream)
	err := s.open(ctx)
	if err != nil {
		return err
	}
	logrus.Infof("Stream opened (identifier: %v) : %v", strconv.Itoa(s.identifier), s.Stream)
	var configurationCtx context.Context
	configurationCtx, s.configurationCancel = context.WithCancel(context.TODO())
	go s.Configuration.WatchStream(configurationCtx)
	return nil
}

// Close the stream in the conduit by unregistering target from the NSP service, stopping
// the stream monitor, and removing the stream from the stream registry.
func (s *Stream) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	logrus.Infof("Close stream: %v", s.Stream)
	if s.configurationCancel != nil {
		s.configurationCancel()
	}
	err := s.TargetRegistry.Unregister(ctx, s.getTarget())
	s.targetStatus = nspAPI.Target_DISABLED
	s.setStatus(ambassadorAPI.StreamStatus_UNAVAILABLE)
	s.identifier = -1
	if err != nil {
		return err
	}
	return nil
}

// Equals checks if the stream is equal to the one in parameter
func (s *Stream) Equals(stream *ambassadorAPI.Stream) bool {
	return s.Stream.Equals(stream)
}

// GetStream returns the current Stream (NSP API struct)
func (s *Stream) GetStream() *ambassadorAPI.Stream {
	return s.Stream
}

// StreamExists sets the availability of the stream in the current trench
func (s *Stream) StreamExists(exists bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.identifier < 0 {
		return nil
	}
	if exists {
		s.setStatus(ambassadorAPI.StreamStatus_OPEN)
	} else {
		s.setStatus(ambassadorAPI.StreamStatus_UNDEFINED)
	}
	return nil
}

func (s *Stream) setIdentifier(exclusionList []string) {
	exclusionListMap := make(map[string]struct{})
	for _, item := range exclusionList {
		exclusionListMap[item] = struct{}{}
	}
	for !s.isIdentifierValid(exclusionListMap, 1, s.MaxNumberOfTargets) {
		rand.Seed(time.Now().UnixNano())
		s.identifier = rand.Intn(s.MaxNumberOfTargets) + 1
	}
}

func (s *Stream) isIdentifierValid(exclusionList map[string]struct{}, min int, max int) bool {
	_, exists := exclusionList[strconv.Itoa(s.identifier)]
	return !exists && s.identifier >= min && s.identifier <= max
}

func (s *Stream) checkIdentifierCollision(identifiersInUse []string) bool {
	found := 0
	for _, identifier := range identifiersInUse {
		if identifier == strconv.Itoa(s.identifier) {
			found++
		}
	}
	return found > 1
}

func (s *Stream) getIdentifiersInUse(ctx context.Context) ([]string, error) {
	identifiers := []string{}
	context, cancel := context.WithCancel(ctx)
	defer cancel()
	targets, err := s.TargetRegistry.GetTargets(context, &nspAPI.Target{
		Status: nspAPI.Target_ANY,
		Type:   nspAPI.Target_DEFAULT,
		Stream: &nspAPI.Stream{
			Conduit: s.Stream.GetConduit().ToNSP(),
		},
	})
	if err != nil {
		return identifiers, err
	}
	for _, target := range targets {
		identifiers = append(identifiers, target.Context[lbTypes.IdentifierKey])
	}
	return identifiers, nil
}

func (s *Stream) getTarget() *nspAPI.Target {
	return &nspAPI.Target{
		Ips: s.ips,
		Context: map[string]string{
			lbTypes.IdentifierKey: strconv.Itoa(s.identifier),
		},
		Status: s.targetStatus,
		Stream: s.Stream.ToNSP(),
	}
}

func (s *Stream) open(ctx context.Context) error {
	identifiersInUse, err := s.getIdentifiersInUse(ctx)
	if err != nil {
		return err
	}
	if len(identifiersInUse) >= s.MaxNumberOfTargets {
		return errors.New("no identifier available to register the target")
	}
	if s.targetStatus != nspAPI.Target_DISABLED {
		return nil
	}
	s.setIdentifier(identifiersInUse)
	err = s.TargetRegistry.Register(ctx, s.getTarget()) // register the target as disabled status
	if err != nil {
		return err
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		identifiersInUse, err := s.getIdentifiersInUse(ctx)
		if err != nil {
			return err
		}
		if len(identifiersInUse) > s.MaxNumberOfTargets {
			err = errors.New("no identifier available to register the target")
			errUnregister := s.TargetRegistry.Unregister(ctx, s.getTarget())
			if errUnregister != nil {
				return fmt.Errorf("%w ; %v", errUnregister, err)
			}
			return err
		}
		// Checks if there is any collision since the last registration/update
		// of the target.
		collision := s.checkIdentifierCollision(identifiersInUse)
		if !collision {
			break
		}
		s.setIdentifier(identifiersInUse)
		err = s.TargetRegistry.Register(ctx, s.getTarget()) // Update the target identifier
		if err != nil {
			return err
		}
	}
	s.targetStatus = nspAPI.Target_ENABLED
	err = s.TargetRegistry.Register(ctx, s.getTarget()) // Update the target as enabled status
	if err != nil {
		return err
	}
	return nil
}

func (s *Stream) setPendingStatus(pendingTrigger <-chan interface{}, cancelPendingCtx context.Context) {
	go func() {
		select {
		case <-cancelPendingCtx.Done():
			return
		case <-pendingTrigger:
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.targetStatus == nspAPI.Target_DISABLED {
				s.setStatus(ambassadorAPI.StreamStatus_UNAVAILABLE)
			}
		}
	}()
}

func (s *Stream) setStatus(status ambassadorAPI.StreamStatus_Status) {
	if s.StreamRegistry == nil {
		return
	}
	if s.pendingCancel != nil {
		s.pendingCancel()
	}
	s.StreamRegistry.SetStatus(s.Stream, status)
}

func DefaultPendingChan() <-chan interface{} {
	pendingChan := make(chan interface{}, 1)
	go func() {
		<-time.After(PendingTime)
		pendingChan <- struct{}{}
	}()
	return pendingChan
}
