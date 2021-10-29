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

package types

import (
	"context"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

const (
	Connected    = 0
	Disconnected = 1
)

type ConduitStatus int

type Conduit interface {
	GetName() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	AddStream(context.Context, Stream) error
	RemoveStream(context.Context, Stream) error
	GetStreams(stream *nspAPI.Stream) []Stream
	GetTrench() Trench
	GetIPs() []string
	SetVIPs(vips []string) error
	Equals(*nspAPI.Conduit) bool
	GetStatus() ConduitStatus
}
