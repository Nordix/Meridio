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

	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
)

// The factory gathers common properties to simplify the
// instanciation of new streams. Mostly useful for the tests.
type StreamFactory interface {
	New(*ambassadorAPI.Stream) (types.Stream, error)
}

// streamManager is responsible for:
// - opening/closing streams based of the streams available in the conduit.
// - Re-opening streams which have been closed by another resource (NSP failures...).
// - setting the status of the streams
type StreamManager interface {
	// AddStream adds the stream to the stream manager, registers it to the
	// stream registry, creates a new stream based on StreamFactory, and open it, if
	// the stream manager is running and if the stream exists in the configuration.
	AddStream(strm *ambassadorAPI.Stream) error
	// RemoveStream removes the stream from the manager, removes it
	// from the stream registry and closes it.
	RemoveStream(context.Context, *ambassadorAPI.Stream) error
	// GetStreams returns the list of streams (opened or not).
	GetStreams() []*ambassadorAPI.Stream
	// Set all streams available in the conduit
	SetStreams([]*nspAPI.Stream)
	// Run open all streams registered and set their
	// status based on the ones available in the conduit.
	Run()
	// Stop closes all streams
	Stop(context.Context) error
}
