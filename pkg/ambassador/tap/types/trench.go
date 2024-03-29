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

//go:generate mockgen -source=trench.go -destination=mocks/trench.go -package=mocks
package types

import (
	"context"

	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
)

// Responsible for connection/disconnecting the conduits, and providing
// a NSP connection to the trench.
type Trench interface {
	Delete(ctx context.Context) error
	// AddConduit creates a conduit and will connect it in background
	AddConduit(context.Context, *ambassadorAPI.Conduit) (Conduit, error)
	// RemoveConduit disconnects and removes the conduit (if existing).
	RemoveConduit(context.Context, *ambassadorAPI.Conduit) error
	GetConduits() []Conduit
	// GetConduit returns the conduit matching to the one in parameter if it exists.
	GetConduit(*ambassadorAPI.Conduit) Conduit
	Equals(*ambassadorAPI.Trench) bool
	GetTrench() *ambassadorAPI.Trench
}
