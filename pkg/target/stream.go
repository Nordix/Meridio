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

package target

import (
	"fmt"
	"os"
	"strconv"

	"github.com/nordix/meridio/pkg/nsp"
)

type Stream struct {
	name          string
	identifier    int
	conduit       *Conduit
	streamWatcher chan<- *StreamEvent
}

type StreamStatus int

const (
	Request = iota
	Close
)

type StreamEvent struct {
	Stream *Stream
	StreamStatus
}

func (s *Stream) Request() error {
	nspClient, err := nsp.NewNetworkServicePlateformClient(s.getNSPService())
	if err != nil {
		return err
	}
	targetContext := map[string]string{
		"identifier": strconv.Itoa(s.identifier),
	}
	err = nspClient.Register(s.conduit.ips, targetContext)
	if err != nil {
		return err
	}
	s.notifyWatcher(Request)
	return nspClient.Delete()
}

func (s *Stream) Delete() error {
	nspClient, err := nsp.NewNetworkServicePlateformClient(s.getNSPService())
	if err != nil {
		return err
	}
	err = nspClient.Unregister(s.conduit.ips)
	if err != nil {
		return err
	}
	s.notifyWatcher(Close)
	return nspClient.Delete()
}

func (s *Stream) notifyWatcher(status StreamStatus) {
	if s.streamWatcher == nil {
		return
	}
	s.streamWatcher <- &StreamEvent{
		Stream:       s,
		StreamStatus: status,
	}
}

func (s *Stream) getNSPService() string {
	return fmt.Sprintf("%s-%s.%s:%d", s.GetConfig().nspServiceName, s.GetTrenchName(), s.GetNamespace(), s.GetConfig().nspServicePort)
}

func (s *Stream) GetName() string {
	return s.name
}

func (s *Stream) GetTrenchName() string {
	return s.conduit.GetTrenchName()
}

func (s *Stream) GetConduitName() string {
	return s.conduit.GetName()
}

func (s *Stream) GetNamespace() string {
	return s.conduit.GetNamespace()
}

func (s *Stream) GetConfig() *Config {
	return s.conduit.GetConfig()
}

func NewStream(name string, conduit *Conduit, streamWatcher chan<- *StreamEvent) *Stream {
	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	stream := &Stream{
		name:          name,
		identifier:    identifier,
		conduit:       conduit,
		streamWatcher: streamWatcher,
	}
	return stream
}
