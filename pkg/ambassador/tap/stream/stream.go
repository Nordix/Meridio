/*
Copyright (c) 2021 Nordix Foundation

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
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	lbTypes "github.com/nordix/meridio/pkg/loadbalancer/types"
)

// Stream implements types.Stream
type Stream struct {
	// Name of the Stream
	Name string
	// Conduit the stream belongs to
	// This should not be nil
	Conduit types.Conduit
	// Channel returning events when the stream is opened/closed
	EventChan chan<- struct{}
	// Maximum number of targets registered in this stream
	MaxNumberOfTargets int
	identifier         int
	status             nspAPI.Target_Status
}

// New is the constructor of Stream.
// The constructor will add the new created stream to the conduit.
func New(
	ctx context.Context,
	name string,
	conduit types.Conduit,
	eventChan chan<- struct{}) (types.Stream, error) {
	stream := &Stream{
		Name:               name,
		identifier:         0,
		Conduit:            conduit,
		EventChan:          eventChan,
		status:             nspAPI.Target_DISABLED,
		MaxNumberOfTargets: maxNumberOfTargets,
	}
	err := conduit.AddStream(ctx, stream)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// Open the stream in the conduit by generating a identifier and registering
// the target to the NSP service while avoiding the identifier collisions.
// If success, no error will be returned and an event will be send via the streamWatcher.
// If not, an error will be returned.
func (s *Stream) Open(ctx context.Context) error {
	identifiersInUse, err := s.getIdentifiersInUse(ctx)
	if err != nil {
		return err
	}
	if len(identifiersInUse) >= s.MaxNumberOfTargets {
		return errors.New("no identifier available to register the target")
	}
	s.setIdentifier(identifiersInUse)
	s.status = nspAPI.Target_DISABLED
	err = s.register(ctx) // register the target as disabled status
	if err != nil {
		return err
	}
	for {
		identifiersInUse, err := s.getIdentifiersInUse(ctx)
		if err != nil {
			return err
		}
		if len(identifiersInUse) > s.MaxNumberOfTargets {
			err = errors.New("no identifier available to register the target")
			errUnregister := s.unregister(ctx)
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
		err = s.update(ctx) // Update the target identifier
		if err != nil {
			return err
		}
	}
	s.status = nspAPI.Target_ENABLED
	err = s.update(ctx) // Update the target as enabled status
	if err != nil {
		return err
	}
	s.notifyWatcher()
	return nil
}

// Close the stream in the conduit by unregistering target from the NSP service.
// If success, no error will be returned and an event will be send via the streamWatcher.
// If not, an error will be returned.
func (s *Stream) Close(ctx context.Context) error {
	err := s.unregister(ctx)
	if err != nil {
		return err
	}
	s.status = nspAPI.Target_DISABLED
	s.notifyWatcher()
	return nil
}

// GetName returns the name of the stream.
func (s *Stream) GetName() string {
	return s.Name
}

// GetConduit returns the conduit the stream belongs to.
func (s *Stream) GetConduit() types.Conduit {
	return s.Conduit
}

func (s *Stream) Equals(stream *nspAPI.Stream) bool {
	if stream == nil {
		return true
	}
	name := true
	if stream.GetName() != "" {
		name = s.GetName() == stream.GetName()
	}
	return name && s.GetConduit().Equals(stream.GetConduit())
}

func (s *Stream) GetStatus() types.StreamStatus {
	if s.status == nspAPI.Target_ENABLED {
		return types.Opened
	}
	return types.Closed
}

func (s *Stream) notifyWatcher() {
	if s.EventChan == nil {
		return
	}
	s.EventChan <- struct{}{}
}

func (s *Stream) register(ctx context.Context) error {
	target := &nspAPI.Target{
		Ips: s.Conduit.GetIPs(),
		Context: map[string]string{
			lbTypes.IdentifierKey: strconv.Itoa(s.identifier),
		},
		Status: s.status,
		Stream: s.getNSPStream(),
	}
	_, err := s.getTargetRegistryClient().Register(ctx, target)
	return err
}

func (s *Stream) update(ctx context.Context) error {
	target := &nspAPI.Target{
		Ips: s.Conduit.GetIPs(),
		Context: map[string]string{
			lbTypes.IdentifierKey: strconv.Itoa(s.identifier),
		},
		Status: s.status,
		Stream: s.getNSPStream(),
	}
	_, err := s.getTargetRegistryClient().Update(ctx, target)
	return err
}

func (s *Stream) unregister(ctx context.Context) error {
	target := &nspAPI.Target{
		Ips:    s.Conduit.GetIPs(),
		Type:   nspAPI.Target_DEFAULT,
		Stream: s.getNSPStream(),
	}
	_, err := s.getTargetRegistryClient().Unregister(ctx, target)
	return err
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
	watchClient, err := s.getTargetRegistryClient().Watch(context, &nspAPI.Target{
		Status: nspAPI.Target_ANY,
		Type:   nspAPI.Target_DEFAULT,
		// Stream: s.getNSPStream(), // todo
		Stream: &nspAPI.Stream{
			Conduit: s.getNSPStream().GetConduit(),
		},
	})
	if err != nil {
		return identifiers, err
	}
	responseTargets, err := watchClient.Recv()
	if err != nil {
		return identifiers, err
	}
	for _, target := range responseTargets.Targets {
		identifiers = append(identifiers, target.Context[lbTypes.IdentifierKey])
	}
	return identifiers, nil
}

func (s *Stream) getTargetRegistryClient() nspAPI.TargetRegistryClient {
	return s.GetConduit().GetTrench().GetTargetRegistryClient()
}

func (s *Stream) getNSPStream() *nspAPI.Stream {
	return &nspAPI.Stream{
		Name: s.Name,
		Conduit: &nspAPI.Conduit{
			Name: s.Conduit.GetName(),
			Trench: &nspAPI.Trench{
				Name: s.Conduit.GetTrench().GetName(),
			},
		},
	}
}
