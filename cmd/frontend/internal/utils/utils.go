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

package utils

import (
	"net"
	"strings"
)

func IsIPv4(ip string) bool {
	return strings.Count(ip, ":") == 0
}

func IsIPv6(ip string) bool {
	return strings.Count(ip, ":") >= 2
}

func StrToIPNet(in string) *net.IPNet {
	if in == "" {
		return nil
	}
	ip, ipNet, err := net.ParseCIDR(in)
	if err != nil {
		return nil
	}
	ipNet.IP = ip
	return ipNet
}

// compare two lists
// return values: (b - a), (a - b)
func Difference(a, b []string) ([]string, []string) {
	m := make(map[string]bool)
	uniqueB := []string{}
	uniqueA := []string{}

	for _, item := range b {
		// items in b
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			//  not in b
			uniqueA = append(uniqueA, item)
		} else {
			// both in a and b; mark that it's not unique to b
			m[item] = false
		}
	}

	// check items unique to b
	for k, v := range m {
		if v {
			uniqueB = append(uniqueB, k)
		}
	}

	return uniqueB, uniqueA
}
