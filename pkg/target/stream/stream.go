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

	nspAPI "github.com/nordix/meridio/api/nsp"
	targetAPI "github.com/nordix/meridio/api/target"
	"github.com/nordix/meridio/pkg/target/types"
)

// Stream implements types.Stream
type Stream struct {
	// Name of the Stream
	Name string
	// Conduit the stream belongs to
	// This should not be nil
	Conduit types.Conduit
	// Channel returning events when the stream is opened/closed
	StreamWatcher chan<- *targetAPI.StreamEvent
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
	streamWatcher chan<- *targetAPI.StreamEvent) (types.Stream, error) {
	stream := &Stream{
		Name:               name,
		identifier:         0,
		Conduit:            conduit,
		StreamWatcher:      streamWatcher,
		status:             nspAPI.Target_Disabled,
		MaxNumberOfTargets: maxNumberOfTargets,
	}
	err := conduit.AddStream(ctx, stream)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// Request the stream in the conduit by generating a identifier and registering
// the target to the NSP service while avoiding the identifier collisions.
// If success, no error will be returned and an event will be send via the streamWatcher.
// If not, an error will be returned.
func (s *Stream) Request(ctx context.Context) error {
	identifiersInUse, err := s.getIdentifiersInUse(ctx)
	if err != nil {
		return err
	}
	if len(identifiersInUse) >= s.MaxNumberOfTargets {
		return errors.New("no identifier available to register the target")
	}
	s.setIdentifier(identifiersInUse)
	s.status = nspAPI.Target_Disabled
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
	s.status = nspAPI.Target_Enabled
	err = s.update(ctx) // Update the target as enabled status
	if err != nil {
		return err
	}
	s.notifyWatcher(targetAPI.StreamEventStatus_Request)
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
	s.status = nspAPI.Target_Disabled
	s.notifyWatcher(targetAPI.StreamEventStatus_Close)
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

func (s *Stream) notifyWatcher(status targetAPI.StreamEventStatus) {
	if s.StreamWatcher == nil {
		return
	}
	s.StreamWatcher <- &targetAPI.StreamEvent{
		Stream: &targetAPI.Stream{
			Conduit: &targetAPI.Conduit{
				NetworkServiceName: s.GetConduit().GetName(),
				Trench: &targetAPI.Trench{
					Name: s.GetConduit().GetTrench().GetName(),
				},
			},
		},
		StreamEventStatus: status,
	}
}

func (s *Stream) register(ctx context.Context) error {
	target := &nspAPI.Target{
		Ips: s.Conduit.GetIPs(),
		Context: map[string]string{
			identifierKey: strconv.Itoa(s.identifier),
		},
		Status: s.status,
	}
	_, err := s.getNSPClient().Register(ctx, target)
	return err
}

func (s *Stream) update(ctx context.Context) error {
	target := &nspAPI.Target{
		Ips: s.Conduit.GetIPs(),
		Context: map[string]string{
			identifierKey: strconv.Itoa(s.identifier),
		},
		Status: s.status,
	}
	_, err := s.getNSPClient().Update(ctx, target)
	return err
}

func (s *Stream) unregister(ctx context.Context) error {
	target := &nspAPI.Target{
		Ips: s.Conduit.GetIPs(),
	}
	_, err := s.getNSPClient().Unregister(ctx, target)
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
	responseTargets, err := s.getNSPClient().GetTargets(ctx, &nspAPI.TargetType{
		Type: nspAPI.Target_DEFAULT,
	})
	if err != nil {
		return identifiers, err
	}
	for _, target := range responseTargets.Targets {
		identifiers = append(identifiers, target.Context[identifierKey])
	}
	return identifiers, nil
}

func (s *Stream) getNSPClient() nspAPI.NetworkServicePlateformServiceClient {
	return s.GetConduit().GetTrench().GetNSPClient()
}
