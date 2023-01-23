/*
Copyright (c) 2023 Nordix Foundation

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

package proxy_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	v1 "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/proxy"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Test_SetIPContext_NSC(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	conduit := &v1.Conduit{
		Name: "",
		Trench: &v1.Trench{
			Name: "",
		},
	}
	ipamClient := &mockIPAM{
		ips: map[ipamAPI.IPFamily][]*ipamAPI.Prefix{
			ipamAPI.IPFamily_IPV4: {
				{
					Address:      "172.16.0.1",
					PrefixLength: 24,
				},
				{
					Address:      "172.16.0.2",
					PrefixLength: 24,
				},
			},
			ipamAPI.IPFamily_IPV6: {
				{
					Address:      "fd00::1",
					PrefixLength: 64,
				},
				{
					Address:      "fd00::2",
					PrefixLength: 64,
				},
			},
		},
		currentIPv4: 0,
		currentIPv6: 0,
	}

	proxy := proxy.Proxy{
		Bridge: nil,
		Subnets: map[ipamAPI.IPFamily]*ipamAPI.Subnet{
			ipamAPI.IPFamily_IPV4: {
				Conduit:  conduit,
				Node:     "Worker",
				IpFamily: ipamAPI.IPFamily_IPV4,
			},
			ipamAPI.IPFamily_IPV6: {
				Conduit:  conduit,
				Node:     "Worker",
				IpFamily: ipamAPI.IPFamily_IPV6,
			},
		},
		IpamClient: ipamClient,
	}

	conn := &networkservice.Connection{}

	err := proxy.SetIPContext(context.TODO(), conn, networking.NSC)
	assert.Nil(t, err)
	assert.NotNil(t, conn.GetContext())
	assert.NotNil(t, conn.GetContext().GetIpContext())
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().DstIpAddrs, []string{"172.16.0.1/24", "fd00::1/64"})
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().SrcIpAddrs, []string{"172.16.0.2/24", "fd00::2/64"})
}

func Test_SetIPContext_NSE(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	conduit := &v1.Conduit{
		Name: "",
		Trench: &v1.Trench{
			Name: "",
		},
	}
	ipamClient := &mockIPAM{
		ips: map[ipamAPI.IPFamily][]*ipamAPI.Prefix{
			ipamAPI.IPFamily_IPV4: {
				{
					Address:      "172.16.0.1",
					PrefixLength: 24,
				},
				{
					Address:      "172.16.0.2",
					PrefixLength: 24,
				},
			},
			ipamAPI.IPFamily_IPV6: {
				{
					Address:      "fd00::1",
					PrefixLength: 64,
				},
				{
					Address:      "fd00::2",
					PrefixLength: 64,
				},
			},
		},
		currentIPv4: 0,
		currentIPv6: 0,
	}
	bridge := &mockBridge{}

	proxy := proxy.Proxy{
		Bridge: bridge,
		Subnets: map[ipamAPI.IPFamily]*ipamAPI.Subnet{
			ipamAPI.IPFamily_IPV4: {
				Conduit:  conduit,
				Node:     "Worker",
				IpFamily: ipamAPI.IPFamily_IPV4,
			},
			ipamAPI.IPFamily_IPV6: {
				Conduit:  conduit,
				Node:     "Worker",
				IpFamily: ipamAPI.IPFamily_IPV6,
			},
		},
		IpamClient: ipamClient,
	}

	conn := &networkservice.Connection{}

	err := proxy.SetIPContext(context.TODO(), conn, networking.NSE)
	assert.Nil(t, err)
	assert.NotNil(t, conn.GetContext())
	assert.NotNil(t, conn.GetContext().GetIpContext())
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().SrcIpAddrs, []string{"172.16.0.1/24", "fd00::1/64"})
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().DstIpAddrs, []string{"172.16.0.2/24", "fd00::2/64"})
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().ExtraPrefixes, []string{"172.16.0.100/24", "fd00::100/64"})
}

func Test_SetIPContext_NSE_Update(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	conduit := &v1.Conduit{
		Name: "",
		Trench: &v1.Trench{
			Name: "",
		},
	}
	ipamClient := &mockIPAM{
		ips: map[ipamAPI.IPFamily][]*ipamAPI.Prefix{
			ipamAPI.IPFamily_IPV4: {
				{
					Address:      "172.16.0.1",
					PrefixLength: 24,
				},
				{
					Address:      "172.16.0.2",
					PrefixLength: 24,
				},
			},
			ipamAPI.IPFamily_IPV6: {
				{
					Address:      "fd00::1",
					PrefixLength: 64,
				},
				{
					Address:      "fd00::2",
					PrefixLength: 64,
				},
			},
		},
		currentIPv4: 0,
		currentIPv6: 0,
	}
	bridge := &mockBridge{}

	proxy := proxy.Proxy{
		Bridge: bridge,
		Subnets: map[ipamAPI.IPFamily]*ipamAPI.Subnet{
			ipamAPI.IPFamily_IPV4: {
				Conduit:  conduit,
				Node:     "Worker",
				IpFamily: ipamAPI.IPFamily_IPV4,
			},
			ipamAPI.IPFamily_IPV6: {
				Conduit:  conduit,
				Node:     "Worker",
				IpFamily: ipamAPI.IPFamily_IPV6,
			},
		},
		IpamClient: ipamClient,
	}

	conn := &networkservice.Connection{
		Id: "abc",
		Context: &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				SrcIpAddrs:    []string{"172.16.0.1/24", "fd00::1/64", "20.0.0.1/32", "2000::1/128"},
				DstIpAddrs:    []string{"172.16.0.2/24", "fd00::2/64"},
				ExtraPrefixes: []string{"172.16.0.100/24", "fd00::100/64"},
				Policies: []*networkservice.PolicyRoute{
					{
						From: "20.0.0.1/32",
						Routes: []*networkservice.Route{
							{
								Prefix:  "0.0.0.0/0",
								NextHop: "172.16.0.100",
							},
						},
					},
					{
						From: "2000::1/128",
						Routes: []*networkservice.Route{
							{
								Prefix:  "::/0",
								NextHop: "fd00::100",
							},
						},
					},
				},
			},
		},
	}

	err := proxy.SetIPContext(context.TODO(), conn, networking.NSE)
	assert.Nil(t, err)
	assert.NotNil(t, conn.GetContext())
	assert.NotNil(t, conn.GetContext().GetIpContext())
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().SrcIpAddrs, []string{"172.16.0.1/24", "fd00::1/64", "20.0.0.1/32", "2000::1/128"})
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().DstIpAddrs, []string{"172.16.0.2/24", "fd00::2/64"})
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().ExtraPrefixes, []string{"172.16.0.100/24", "fd00::100/64"})
}

func Test_SetIPContext_NSE_Update_New_IPs(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	conduit := &v1.Conduit{
		Name: "",
		Trench: &v1.Trench{
			Name: "",
		},
	}
	ipamClient := &mockIPAM{
		ips: map[ipamAPI.IPFamily][]*ipamAPI.Prefix{
			ipamAPI.IPFamily_IPV4: {
				{
					Address:      "172.16.0.10",
					PrefixLength: 24,
				},
				{
					Address:      "172.16.0.20",
					PrefixLength: 24,
				},
			},
			ipamAPI.IPFamily_IPV6: {
				{
					Address:      "fd00::10",
					PrefixLength: 64,
				},
				{
					Address:      "fd00::20",
					PrefixLength: 64,
				},
			},
		},
		currentIPv4: 0,
		currentIPv6: 0,
	}
	bridge := &mockBridge{}

	proxy := proxy.Proxy{
		Bridge: bridge,
		Subnets: map[ipamAPI.IPFamily]*ipamAPI.Subnet{
			ipamAPI.IPFamily_IPV4: {
				Conduit:  conduit,
				Node:     "Worker",
				IpFamily: ipamAPI.IPFamily_IPV4,
			},
			ipamAPI.IPFamily_IPV6: {
				Conduit:  conduit,
				Node:     "Worker",
				IpFamily: ipamAPI.IPFamily_IPV6,
			},
		},
		IpamClient: ipamClient,
	}

	conn := &networkservice.Connection{
		Id: "abc",
		Context: &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				SrcIpAddrs:    []string{"172.16.0.1/24", "fd00::1/64", "20.0.0.1/32", "2000::1/128"},
				DstIpAddrs:    []string{"172.16.0.2/24", "fd00::2/64"},
				ExtraPrefixes: []string{"172.16.0.10/24", "fd00::10/64"},
				Policies: []*networkservice.PolicyRoute{
					{
						From: "20.0.0.1/32",
						Routes: []*networkservice.Route{
							{
								Prefix:  "0.0.0.0/0",
								NextHop: "172.16.0.10",
							},
						},
					},
					{
						From: "2000::1/128",
						Routes: []*networkservice.Route{
							{
								Prefix:  "::/0",
								NextHop: "fd00::10",
							},
						},
					},
				},
			},
		},
	}

	err := proxy.SetIPContext(context.TODO(), conn, networking.NSE)
	assert.Nil(t, err)
	assert.NotNil(t, conn.GetContext())
	assert.NotNil(t, conn.GetContext().GetIpContext())
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().SrcIpAddrs, []string{"172.16.0.10/24", "fd00::10/64", "20.0.0.1/32", "2000::1/128"})
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().DstIpAddrs, []string{"172.16.0.20/24", "fd00::20/64"})
	assert.ElementsMatch(t, conn.GetContext().GetIpContext().ExtraPrefixes, []string{"172.16.0.100/24", "fd00::100/64"})
	policies := conn.GetContext().GetIpContext().Policies
	assert.Len(t, policies, 2)
	if policies[0].From == "20.0.0.1/32" {
		assert.Equal(t, "20.0.0.1/32", policies[0].From)
		assert.Len(t, policies[0].Routes, 1)
		assert.Equal(t, "0.0.0.0/0", policies[0].Routes[0].Prefix)
		assert.Equal(t, "172.16.0.100", policies[0].Routes[0].NextHop)
		assert.Equal(t, "2000::1/128", policies[1].From)
		assert.Len(t, policies[1].Routes, 1)
		assert.Equal(t, "::/0", policies[1].Routes[0].Prefix)
		assert.Equal(t, "fd00::100", policies[1].Routes[0].NextHop)
	} else {
		assert.Equal(t, "20.0.0.1/32", policies[1].From)
		assert.Len(t, policies[1].Routes, 1)
		assert.Equal(t, "0.0.0.0/0", policies[1].Routes[0].Prefix)
		assert.Equal(t, "172.16.0.100", policies[1].Routes[0].NextHop)
		assert.Equal(t, "2000::1/128", policies[0].From)
		assert.Len(t, policies[0].Routes, 1)
		assert.Equal(t, "::/0", policies[0].Routes[0].Prefix)
		assert.Equal(t, "fd00::100", policies[0].Routes[0].NextHop)
	}
}

type mockBridge struct {
	kernel.Bridge
}

func (mb *mockBridge) GetLocalPrefixes() []string {
	return []string{"172.16.0.100/24", "fd00::100/64"}
}

type mockIPAM struct {
	ips         map[ipamAPI.IPFamily][]*ipamAPI.Prefix
	currentIPv4 int
	currentIPv6 int
}

func (mi *mockIPAM) Allocate(ctx context.Context, in *ipamAPI.Child, opts ...grpc.CallOption) (*ipamAPI.Prefix, error) {
	if in.Subnet.IpFamily == ipamAPI.IPFamily_IPV4 {
		res := mi.ips[in.Subnet.IpFamily][mi.currentIPv4]
		mi.currentIPv4++
		return res, nil
	}
	res := mi.ips[in.Subnet.IpFamily][mi.currentIPv6]
	mi.currentIPv6++
	return res, nil
}

func (mi *mockIPAM) Release(ctx context.Context, in *ipamAPI.Child, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}
