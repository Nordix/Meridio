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

import (
	"context"
	"net/url"
)

// Option is an option pattern for NewChecker
type Option func(o *checkerOptions)

// WithCtx sets context
func WithCtx(ctx context.Context) Option {
	return func(o *checkerOptions) {
		o.ctx = ctx
	}
}

// WithURL sets url
func WithURL(u *url.URL) Option {
	return func(o *checkerOptions) {
		o.u = u
	}
}

type checkerOptions struct {
	ctx context.Context
	u   *url.URL
}
