/*
Copyright (c) 2021 Nordix Foundation
Copyright (c) 2024 OpenInfra Foundation Europe

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
)

type Trench interface {
	Prefix
	GetConduit(ctx context.Context, name string) (Conduit, error)
}

type Conduit interface {
	Prefix
	GetNode(ctx context.Context, name string) (Node, error)
	RemoveNode(ctx context.Context, name string) error
}

type Node interface {
	Prefix
	Allocate(ctx context.Context, name string) (Prefix, error)
	Release(ctx context.Context, name string) error
}

type Prefix interface {
	GetName() string
	GetCidr() string
	GetParent() Prefix
	Equals(Prefix) bool
}

type Storage interface {
	Add(ctx context.Context, prefix Prefix) error
	Update(ctx context.Context, prefix Prefix) error
	Delete(ctx context.Context, prefix Prefix) error
	Get(ctx context.Context, name string, parent Prefix) (Prefix, error)
	GetChilds(ctx context.Context, prefix Prefix) ([]Prefix, error)
}
