/*
Copyright (c) 2021-2023 Nordix Foundation

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

package probe

import "time"

type Option func(o *probeOptions)

// WithAddress sets address for probe
func WithAddress(addr string) Option {
	return func(o *probeOptions) {
		o.addr = addr
	}
}

// WithService sets service for probe
func WithService(service string) Option {
	return func(o *probeOptions) {
		o.service = service
	}
}

// WithSpiffe sets spiffe option for probe
func WithSpiffe() Option {
	return func(o *probeOptions) {
		o.spiffe = true
	}
}

// WithRPCTimeout sets RPC timeout for probe
func WithRPCTimeout(rpcTimeout time.Duration) Option {
	return func(o *probeOptions) {
		o.rpcTimeout = rpcTimeout
	}
}

// WithConnectTimeout sests connect timeout probe
func WithConnectTimeout(connTimeout time.Duration) Option {
	return func(o *probeOptions) {
		o.connTimeout = connTimeout
	}
}

// WithUserAgent sests user agent
func WithUserAgent(userAgent string) Option {
	return func(o *probeOptions) {
		o.userAgent = userAgent
	}
}

type probeOptions struct {
	userAgent   string
	addr        string
	service     string
	connTimeout time.Duration
	rpcTimeout  time.Duration
	spiffe      bool
}
