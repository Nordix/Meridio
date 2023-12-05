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

package stream

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/go-logr/logr"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	lbTypes "github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/log"
)

// Stream implements types.Stream (/pkg/ambassador/tap/types)
// has
type Stream struct {
	Stream         *ambassadorAPI.Stream
	TargetRegistry TargetRegistry
	// Contains Function to get the IPs of the conduit
	Conduit Conduit
	// Maximum number of targets registered in this stream
	identifier   int
	targetStatus nspAPI.Target_Status
	// When opening the stream, the IPs are save, so, if the IPs of the conduit are changed
	// before closing, this IP list will be used.
	ips    []string
	mu     sync.Mutex
	logger logr.Logger
}

// New is the constructor of Stream.
// Can return an error if the stream is invalid.
func New(stream *ambassadorAPI.Stream,
	targetRegistryClient nspAPI.TargetRegistryClient,
	conduit Conduit) (*Stream, error) {
	logger := log.Logger.WithValues("class", "Stream", "stream", stream)
	logger.Info("Create stream")
	// todo: check if stream valid
	s := &Stream{
		Stream:         stream,
		TargetRegistry: newTargetRegistryImpl(targetRegistryClient),
		Conduit:        conduit,
		identifier:     -1,
		targetStatus:   nspAPI.Target_DISABLED,
		logger:         logger,
	}
	return s, nil
}

// Open the stream in the conduit by generating a identifier and registering
// the target to the NSP service while avoiding the identifier collisions.
// If success, no error will be returned.
func (s *Stream) Open(ctx context.Context, nspStream *nspAPI.Stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ips = s.Conduit.GetIPs()
	if len(s.ips) <= 0 {
		return errors.New("ips not set")
	}
	if s.targetStatus == nspAPI.Target_ENABLED && s.isIdentifierInRange(1, int(nspStream.GetMaxTargets())) {
		return s.refresh(ctx, nspStream)
	}
	s.logger.Info("Attempt to open stream")
	err := s.open(ctx, nspStream)
	if err != nil {
		return err
	}
	s.logger.Info("Stream opened", "identifier", s.identifier, "target", s.getTarget())
	return nil
}

// Close the stream in the conduit by unregistering target from the NSP service.
func (s *Stream) Close(ctx context.Context) error {
	s.mu.Lock()
	defer func() {
		s.mu.Unlock()
		s.targetStatus = nspAPI.Target_DISABLED
		s.identifier = -1
	}()
	if s.identifier < 0 {
		// Avoid spamming TargetRegistry with Unregister request that makes no sense.
		// Meaning what would be the point of unregistering using non-existent identifier?
		// Note: Currently Close() is called on every setStream configuration event
		// if the stream is not part of the configuration (either removed from it
		// before or was never added).
		return nil
	}
	s.logger.Info("Close stream")
	target := s.getTarget()
	err := s.TargetRegistry.Unregister(ctx, target)
	if err != nil {
		return fmt.Errorf("failed to unregister target %v: %w", target, err)
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

func (s *Stream) setIdentifier(exclusionList []string, maxTargets int) {
	exclusionListMap := make(map[string]struct{})
	for _, item := range exclusionList {
		exclusionListMap[item] = struct{}{}
	}
	for !s.isIdentifierValid(exclusionListMap, 1, maxTargets) {
		rand.New(rand.NewSource(time.Now().UnixNano()))
		s.identifier = rand.Intn(maxTargets) + 1
	}
}

func (s *Stream) isIdentifierValid(exclusionList map[string]struct{}, min int, max int) bool {
	_, exists := exclusionList[strconv.Itoa(s.identifier)]
	return !exists && s.isIdentifierInRange(min, max)
}

func (s *Stream) isIdentifierInRange(min int, max int) bool {
	return s.identifier >= min && s.identifier <= max
}

func (s *Stream) checkIdentifierCollision(identifiersInUse []string) bool {
	for _, identifier := range identifiersInUse {
		if identifier == strconv.Itoa(s.identifier) {
			return true
		}
	}
	return false
}

// getIdentifiersInUse except current one
func (s *Stream) getIdentifiersInUse(ctx context.Context) ([]string, error) {
	identifiers := []string{}
	targets, err := s.TargetRegistry.GetTargets(ctx, &nspAPI.Target{
		Status: nspAPI.Target_ANY,
		Type:   nspAPI.Target_DEFAULT,
		Stream: s.Stream.ToNSP(),
	})
	if err != nil {
		return identifiers, fmt.Errorf("failed to get targets for identifiers: %w", err)
	}
	for _, target := range targets {
		if sameIps(target.GetIps(), s.ips) {
			continue
		}
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

func (s *Stream) open(ctx context.Context, nspStream *nspAPI.Stream) error {
	if s.targetStatus != nspAPI.Target_DISABLED {
		return nil
	}
	// Get identifiers in use (it includes the enabled and disabled entries)
	identifiersInUse, err := s.getIdentifiersInUse(ctx)
	if err != nil {
		return err
	}
	// Check if any identifier is available to be registered with
	// TODO: verify if all targets have a valid identifier, some might have empty context or an
	// invalid identifier, so some (identifiers) might still be available.
	if len(identifiersInUse) >= int(nspStream.GetMaxTargets()) {
		return errors.New("no identifier available to register the target")
	}
	if s.identifier <= 0 {
		s.setIdentifier(identifiersInUse, int(nspStream.GetMaxTargets()))
	}
	// register the target as disabled status
	err = s.TargetRegistry.Register(ctx, s.getTarget())
	if err != nil {
		return fmt.Errorf("failed to disable status of target %v: %w", s.getTarget(), err)
	}
	for {
		if ctx.Err() != nil {
			return fmt.Errorf("context error during open: %w", ctx.Err())
		}
		// Get again the identifiers to check if there is any collisions
		identifiersInUse, err := s.getIdentifiersInUse(ctx)
		if err != nil {
			return fmt.Errorf("cannot do collision check for target %v: %w", s.getTarget(), err)
		}
		// Check if any identifier is available to be registered with
		if len(identifiersInUse) >= int(nspStream.GetMaxTargets()) {
			// All identifiers are taken including ours, unregister the target.
			err = errors.New("no identifier available to register the target")
			errUnregister := s.TargetRegistry.Unregister(ctx, s.getTarget())
			if errUnregister != nil {
				return fmt.Errorf("%w ; %w", errUnregister, err)
			}
			return err
		}
		// Checks if there is any collision since the last registration/update
		// of the target.
		collision := s.checkIdentifierCollision(identifiersInUse)
		if !collision {
			break
		}
		// Unregister target with identifier collision (release the offending identifier)
		collidingTarget := s.getTarget()
		if err := s.TargetRegistry.Unregister(ctx, collidingTarget); err != nil {
			s.logger.Info("Did not manage to unregister colliding target", "error", err, "target", collidingTarget)
		}
		// Update the target with a new available identifier
		// (Remember, there was a collision yet the number of other identifiers
		// in use did not reach the maxTargets limit. There must be at least one
		// available based on the last fetched list of identifiers.)
		s.setIdentifier(identifiersInUse, int(nspStream.GetMaxTargets()))
		err = s.TargetRegistry.Register(ctx, s.getTarget())
		if err != nil {
			return fmt.Errorf("failed to update identifier of target %v: %w", s.getTarget(), err)
		}
	}
	s.targetStatus = nspAPI.Target_ENABLED
	// Update the target as enabled status
	err = s.TargetRegistry.Register(ctx, s.getTarget())
	if err != nil {
		return fmt.Errorf("failed to enable status of target %v: %w", s.getTarget(), err)
	}
	return nil
}

func (s *Stream) refresh(ctx context.Context, nspStream *nspAPI.Stream) error {
	target := s.getTarget()
	err := s.TargetRegistry.Register(ctx, target)
	if err != nil {
		return fmt.Errorf("failed to refresh target %v: %w", target, err)
	}
	targets, err := s.TargetRegistry.GetTargets(ctx, &nspAPI.Target{
		Status: nspAPI.Target_ANY,
		Type:   nspAPI.Target_DEFAULT,
		Stream: s.Stream.ToNSP(),
	})
	if err != nil {
		return fmt.Errorf("refresh cannot be verified by registry: %w", err)
	}
	ips := s.getTarget().GetIps()
	for _, target := range targets {
		if !sameIps(ips, target.GetIps()) {
			continue
		}
		// found current target
		// if target is enabled then everything is fine
		if target.Status == nspAPI.Target_ENABLED {
			return nil
		}
		break
	}
	// Target is disabled since the NSP has set its status to disable
	// during refresh. This happened since the NSP has not received the
	// refresh on time, so it has removed the target. When the NSP finnally
	// received the refresh (register call), it considered it as a new registration
	// and then has overwritten the status to DISABLED (it is not possible to register
	// a target as enabled, the target has to be registered to DISABLED, and then
	// updated to ENABLED). The target then has to call open function.
	s.targetStatus = nspAPI.Target_DISABLED
	if err := s.open(ctx, nspStream); err != nil {
		return fmt.Errorf("refresh failed to re-open stream: %w", err)
	}
	s.logger.Info("Stream re-opened during refresh", "identifier", s.identifier, "target", s.getTarget())
	return nil
}

// note: fails on {"1", "2", "3"}, {"1", "2", "2"}
func sameIps(ipsA []string, ipsB []string) bool {
	if len(ipsA) != len(ipsB) {
		return false
	}
	ipMap := map[string]interface{}{}
	for _, ip := range ipsA {
		ipMap[ip] = struct{}{}
	}
	for _, ip := range ipsB {
		_, exists := ipMap[ip]
		if !exists {
			return false
		}
	}
	return true
}
