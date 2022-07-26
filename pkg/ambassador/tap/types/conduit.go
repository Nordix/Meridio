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

//go:generate mockgen -source=conduit.go -destination=mocks/conduit.go -package=mocks
package types

import (
	"context"

	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
)

const (
	Connected    = 0
	Disconnected = 1
)

type ConduitStatus int

// Responsible for requesting/closing the NSM Connection to the conduit,
// managing the streams and configuring the VIPs.
type Conduit interface {
	// Connect requests the connection to NSM and, if success, open all streams added
	// and confiure the VIPs
	Connect(ctx context.Context) error
	// Disconnect closes the connection from NSM, closes all streams
	// and removes the VIP configuration
	Disconnect(ctx context.Context) error
	// AddStream creates a stream and will open it in background
	AddStream(context.Context, *ambassadorAPI.Stream) error
	// RemoveStream closes and removes the stream (if existing)
	RemoveStream(context.Context, *ambassadorAPI.Stream) error
	GetStreams() []*ambassadorAPI.Stream
	Equals(*ambassadorAPI.Conduit) bool
	GetConduit() *ambassadorAPI.Conduit
}
