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

package prefix_test

import (
	"net"
	"testing"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/stretchr/testify/assert"
)

type testOverlappingPrefixes struct {
	cidr1  string
	cidr2  string
	result bool
}

func Test_OverlappingPrefixes(t *testing.T) {
	tests := []*testOverlappingPrefixes{
		{
			cidr1:  "",
			cidr2:  "",
			result: false,
		},
		{
			cidr1:  "abc",
			cidr2:  "abc",
			result: false,
		},
		{
			cidr1:  "169.16.0.0/24",
			cidr2:  "169.16.1.0/24",
			result: false,
		},
		{
			cidr1:  "169.16.0.0/24",
			cidr2:  "169.16.1.0/16",
			result: true,
		},
		{
			cidr1:  "0.0.0.0/0",
			cidr2:  "169.16.1.0/16",
			result: true,
		},
		{
			cidr1:  "169.16.1.0/16",
			cidr2:  "169.16.1.0/16",
			result: true,
		},
		{
			cidr1:  "2001:db8:0:0::/64",
			cidr2:  "2001:db8:0:1::/64",
			result: false,
		},
		{
			cidr1:  "2001:db8:0:0::/64",
			cidr2:  "2001:db8:0:1::/32",
			result: true,
		},
		{
			cidr1:  "::/0",
			cidr2:  "2001:db8:0:1::/32",
			result: true,
		},
		{
			cidr1:  "2001:db8:0:1::/32",
			cidr2:  "2001:db8:0:1::/32",
			result: true,
		},
	}
	for _, test := range tests {
		assert.Equal(t, test.result, prefix.OverlappingPrefixes(test.cidr1, test.cidr2))
	}
}

type testNextPrefix struct {
	prefix *net.IPNet
	result *net.IPNet
}

func Test_NextPrefix(t *testing.T) {
	_, p1, _ := net.ParseCIDR("169.16.0.0/24")
	_, r1, _ := net.ParseCIDR("169.16.1.0/24")
	_, p2, _ := net.ParseCIDR("169.15.0.0/16")
	_, r2, _ := net.ParseCIDR("169.16.0.0/16")
	_, p3, _ := net.ParseCIDR("169.16.0.0/31")
	_, r3, _ := net.ParseCIDR("169.16.0.2/31")
	_, p4, _ := net.ParseCIDR("169.16.0.0/32")
	_, r4, _ := net.ParseCIDR("169.16.0.1/32")
	_, p5, _ := net.ParseCIDR("169.16.0.255/32")
	_, r5, _ := net.ParseCIDR("169.16.1.0/32")
	_, p6, _ := net.ParseCIDR("255.255.255.0/24")
	_, r6, _ := net.ParseCIDR("0.0.0.0/24")
	_, p7, _ := net.ParseCIDR("169.16.1.0/25")
	_, r7, _ := net.ParseCIDR("169.16.1.128/25")
	_, p8, _ := net.ParseCIDR("2001:db8:0:0::/64")
	_, r8, _ := net.ParseCIDR("2001:db8:0:1::/64")
	_, p9, _ := net.ParseCIDR("2001:1::/32")
	_, r9, _ := net.ParseCIDR("2001:2::/32")
	_, p10, _ := net.ParseCIDR("2001:1::9/127")
	_, r10, _ := net.ParseCIDR("2001:1::a/127")
	_, p11, _ := net.ParseCIDR("2001:1::0/128")
	_, r11, _ := net.ParseCIDR("2001:1::1/128")
	_, p12, _ := net.ParseCIDR("2001:1::ffff/128")
	_, r12, _ := net.ParseCIDR("2001:1::1:0000/128")
	_, p13, _ := net.ParseCIDR("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128")
	_, r13, _ := net.ParseCIDR("::/128")
	_, p14, _ := net.ParseCIDR("2001:0100::/25")
	_, r14, _ := net.ParseCIDR("2001:0180::/25")
	_, p15, _ := net.ParseCIDR("2001:1::ff/128")
	_, r15, _ := net.ParseCIDR("2001:1::0100/128")
	tests := []*testNextPrefix{
		{prefix: p1, result: r1},
		{prefix: p2, result: r2},
		{prefix: p3, result: r3},
		{prefix: p4, result: r4},
		{prefix: p5, result: r5},
		{prefix: p6, result: r6},
		{prefix: p7, result: r7},
		{prefix: p8, result: r8},
		{prefix: p9, result: r9},
		{prefix: p10, result: r10},
		{prefix: p11, result: r11},
		{prefix: p12, result: r12},
		{prefix: p13, result: r13},
		{prefix: p14, result: r14},
		{prefix: p15, result: r15},
	}
	for _, test := range tests {
		assert.Equal(t, test.result, prefix.NextPrefix(test.prefix))
	}
}

type testLastIP struct {
	prefix *net.IPNet
	result net.IP
}

func Test_LastIP(t *testing.T) {
	_, p1, _ := net.ParseCIDR("169.16.0.0/24")
	r1 := net.ParseIP("169.16.0.255")
	tests := []*testLastIP{
		{prefix: p1, result: r1},
	}
	for _, test := range tests {
		assert.Equal(t, test.result.String(), prefix.LastIP(test.prefix).String())
	}
}
