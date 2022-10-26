/*
Copyright (c) 2022 Nordix Foundation
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

package main

import (
	"errors"
	"math"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

type IdentifierOffsetGenerator struct {
	Start   int
	Streams map[*nspAPI.Stream]int
	mu      sync.Mutex
}

// NewIdentifierOffsetGenerator is the constructor of the IdentifierOffsetGenerator struct
// It sets the Start property with the one pass as parameter and initialize the streams map
func NewIdentifierOffsetGenerator(start int) *IdentifierOffsetGenerator {
	identifierOffsetGenerator := &IdentifierOffsetGenerator{
		Start:   start,
		Streams: map[*nspAPI.Stream]int{},
	}
	return identifierOffsetGenerator
}

// Generate returns an unused identifier offset for the routing tables to not collide with the
// ones of the other streams. The offset for each streams are saved in memory, so if a stream
// already requested an offset, the saved one will be returned.
func (iog *IdentifierOffsetGenerator) Generate(stream *nspAPI.Stream) (int, error) {
	iog.mu.Lock()
	defer iog.mu.Unlock()
	offset, exists := iog.get(stream)
	if exists {
		return offset, nil
	}
	offset = iog.Start
search:
	for {
		if offset >= (math.MaxInt - int(stream.GetMaxTargets()) + 1) {
			return 0, errors.New("unable to generate identifier offset")
		}
		for s, os := range iog.Streams {
			sStart := os
			sEnd := os + int(s.GetMaxTargets()) - 1
			streamStart := offset
			streamEnd := offset + int(stream.GetMaxTargets()) - 1
			if streamStart <= sEnd && streamEnd >= sStart {
				offset = os + int(s.GetMaxTargets())
				continue search
			}
		}
		break
	}
	iog.Streams[stream] = offset
	return offset, nil
}

// Release allows the streams to free the identifier offset no longer used.
func (iog *IdentifierOffsetGenerator) Release(streamName string) {
	iog.mu.Lock()
	defer iog.mu.Unlock()
	for s := range iog.Streams {
		if streamName == s.GetName() {
			delete(iog.Streams, s)
			return
		}
	}
}

func (iog *IdentifierOffsetGenerator) get(stream *nspAPI.Stream) (int, bool) {
	for s, os := range iog.Streams {
		if stream.GetName() == s.GetName() {
			return os, true
		}
	}
	return -1, false
}
