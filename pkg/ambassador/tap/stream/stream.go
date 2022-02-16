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
	lbTypes "github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/sirupsen/logrus"
)

// Stream implements types.Stream (/pkg/ambassador/tap/types)
// has
type Stream struct {
	Stream         *ambassadorAPI.Stream
	TargetRegistry TargetRegistry
	// Contains Function to get the IPs of the conduit
	Conduit Conduit
	// Maximum number of targets registered in this stream
	MaxNumberOfTargets int
	identifier         int
	targetStatus       nspAPI.Target_Status
	// When opening the stream, the IPs are save, so, if the IPs of the conduit are changed
	// before closing, this IP list will be used.
	ips []string
	mu  sync.Mutex
}

// New is the constructor of Stream.
// Can return an error if the stream is invalid.
func New(stream *ambassadorAPI.Stream,
	targetRegistryClient nspAPI.TargetRegistryClient,
	maxNumberOfTargets int,
	conduit Conduit) (*Stream, error) {
	// todo: check if stream valid
	s := &Stream{
		Stream:             stream,
		TargetRegistry:     newTargetRegistryImpl(targetRegistryClient),
		Conduit:            conduit,
		MaxNumberOfTargets: maxNumberOfTargets,
		identifier:         -1,
		targetStatus:       nspAPI.Target_DISABLED,
	}
	return s, nil
}

// Open the stream in the conduit by generating a identifier and registering
// the target to the NSP service while avoiding the identifier collisions.
// If success, no error will be returned.
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
	return nil
}

// Close the stream in the conduit by unregistering target from the NSP service.
func (s *Stream) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	logrus.Infof("Close stream: %v", s.Stream)
	err := s.TargetRegistry.Unregister(ctx, s.getTarget())
	s.targetStatus = nspAPI.Target_DISABLED
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
	// Get identifiers in use (it includes the enabled and disabled entries)
	identifiersInUse, err := s.getIdentifiersInUse(ctx)
	if err != nil {
		return err
	}
	// Check if any identifier is available to be registered with
	if len(identifiersInUse) >= s.MaxNumberOfTargets {
		return errors.New("no identifier available to register the target")
	}
	if s.targetStatus != nspAPI.Target_DISABLED {
		return nil
	}
	s.setIdentifier(identifiersInUse)
	// register the target as disabled status
	err = s.TargetRegistry.Register(ctx, s.getTarget())
	if err != nil {
		return err
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Get again the identifiers to check if there is any collisions
		identifiersInUse, err := s.getIdentifiersInUse(ctx)
		if err != nil {
			return err
		}
		// Check if any identifier is available to be registered with
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
		// Update the target with a new available identifier
		s.setIdentifier(identifiersInUse)
		err = s.TargetRegistry.Register(ctx, s.getTarget())
		if err != nil {
			return err
		}
	}
	s.targetStatus = nspAPI.Target_ENABLED
	// Update the target as enabled status
	err = s.TargetRegistry.Register(ctx, s.getTarget())
	if err != nil {
		return err
	}
	return nil
}
