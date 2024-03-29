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

//go:generate mockgen -source=stream.go -destination=mocks/stream.go -package=mocks
package types

import (
	"context"

	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

const (
	Opened = 0
	Closed = 1
)

type StreamStatus int

// Stream is an interface that rate limits items being added to the queue.
type Stream interface {
	// Open the stream in the conduit by generating a identifier and registering
	// the target to the NSP service while avoiding the identifier collisions.
	Open(ctx context.Context, nspStream *nspAPI.Stream) error
	// Close the stream in the conduit by unregistering target from the NSP service.
	Close(ctx context.Context) error
	Equals(*ambassadorAPI.Stream) bool
	GetStream() *ambassadorAPI.Stream
}
