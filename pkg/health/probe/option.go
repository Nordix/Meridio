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

package probe

import "fmt"

// Option is an option pattern for GrpcHealthProbe
type Option func(o *probeOptions)

// WithCommand sets grpc health probe command name
func WithCommand(cmd string) Option {
	return func(o *probeOptions) {
		o.cmd = cmd
	}
}

// WithAddress sets address for grpc health probe
func WithAddress(addr string) Option {
	return func(o *probeOptions) {
		o.addr = fmt.Sprintf("-addr=%v", addr)
	}
}

// WithService sets service for grpc health probe
func WithService(service string) Option {
	return func(o *probeOptions) {
		o.service = fmt.Sprintf("-service=%v", service)
	}
}

// WithSpiffe sets spiffe option for grpc health probe
func WithSpiffe() Option {
	return func(o *probeOptions) {
		o.spiffe = "-spiffe"
	}
}

// WithRPCTimeout RPC timeout for grpc health probe
func WithRPCTimeout(rpcTimeout string) Option {
	return func(o *probeOptions) {
		o.rpcTimeout = fmt.Sprintf("-rpc-timeout=%v", rpcTimeout)
	}
}

// WithConnectTimeout connect timeout for grpc health probe
func WithConnectTimeout(connTimeout string) Option {
	return func(o *probeOptions) {
		o.connTimeout = fmt.Sprintf("-connect-timeout=%v", connTimeout)
	}
}

type probeOptions struct {
	cmd         string
	addr        string
	service     string
	connTimeout string
	rpcTimeout  string
	spiffe      string
}
