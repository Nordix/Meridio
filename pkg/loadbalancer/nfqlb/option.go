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

package nfqlb

import "github.com/google/nftables"

type Option func(o *nfoptions)

type nfoptions struct {
	nfqueue string
	table   *nftables.Table
	fanout  bool
}

func WithNFQueue(queue string) Option {
	return func(o *nfoptions) {
		o.nfqueue = queue
	}
}

func WithNFQueueFanout(fanout bool) Option {
	return func(o *nfoptions) {
		o.fanout = fanout
	}
}

type LbOption func(o *lbOptions)

type lbOptions struct {
	name string
	m    int
	n    int
}

func WithLbName(name string) LbOption {
	return func(o *lbOptions) {
		o.name = name
	}
}

func WithMaglevM(m int) LbOption {
	return func(o *lbOptions) {
		o.m = m
	}
}

func WithMaglevN(n int) LbOption {
	return func(o *lbOptions) {
		o.n = n
	}
}
