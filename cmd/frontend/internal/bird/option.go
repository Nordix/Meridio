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

package bird

// Option is an option pattern for NewProtocol
type Option func(o *protoOptions)

// WithProtocolName sets protocol name
func WithName(name string) Option {
	return func(o *protoOptions) {
		o.m[protoName] = name
	}
}

// WithProtocolName sets protocol type (e.g. BGP, Static)
func WithProto(proto string) Option {
	return func(o *protoOptions) {
		o.m[protoProto] = proto
	}
}

// WithState sets protocol state (e.g. up)
func WithState(state string) Option {
	return func(o *protoOptions) {
		o.m[protoState] = state
	}
}

// WithInfo sets protocol info (e.g. Established)
func WithInfo(info string) Option {
	return func(o *protoOptions) {
		o.m[protoInfo] = info
	}
}

// WithInterface sets protocol interface (e.g. ext-vlan)
func WithInterface(itf string) Option {
	return func(o *protoOptions) {
		o.m[protoItf] = itf
	}
}

// WithNeighbor sets protocol neighbor (e.g. 169.254.100.254)
func WithNeighbor(ip string) Option {
	return func(o *protoOptions) {
		o.m[protoNbr] = ip
	}
}

// WithBfdSessions sets known bfd sessions
func WithBfdSessions(bfd string) Option {
	return func(o *protoOptions) {
		o.m[bfdSessions] = bfd
	}
}

// WithOutLog sets out log
func WithOutLog(log *string) Option {
	return func(o *protoOptions) {
		o.log = log
	}
}

type protoOptions struct {
	m   protocolMap
	log *string
}
