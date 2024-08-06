/*
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

package sqlite

import "context"

const (
	lifetimeKey contextKeyType = "expirable"
)

type contextKeyType string

// WithExpirable -
// Stores in context whether the sqlite record must be expirable
func WithExpirable(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, lifetimeKey, struct{}{})
}

// Expirable -
// Returns from context if sqlite record is expirable
func Expirable(ctx context.Context) bool {
	_, ok := ctx.Value(lifetimeKey).(struct{})
	return ok
}
