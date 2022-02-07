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

package types

import (
	"context"

	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
)

type Trench interface {
	Delete(ctx context.Context) error
	AddConduit(context.Context, *ambassadorAPI.Conduit) (Conduit, error)
	RemoveConduit(context.Context, *ambassadorAPI.Conduit) error
	GetConduits() []Conduit
	GetConduit(*ambassadorAPI.Conduit) Conduit
	Equals(*ambassadorAPI.Trench) bool
	GetTrench() *ambassadorAPI.Trench
}
