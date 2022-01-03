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

package health

import "context"

const (
	healthServerKey contextKeyType = "healthServer"
)

type contextKeyType string

// withHealthServer -
// Store Health Server (Checker) in Context
func WithHealthServer(parent context.Context, server *Checker) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, healthServerKey, server)
}

// Server -
// Returns Health Server (Checker) from the context.Context
func HealthServer(ctx context.Context) *Checker {
	rv, ok := ctx.Value(healthServerKey).(*Checker)
	if ok && rv != nil {
		return rv
	}
	return nil
}
