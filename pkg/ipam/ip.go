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

package ipam

import (
	"net"
)

type IPFamily uint8

const (
	IPv4 = 0
	IPv6 = 1
)

func IsCIDR(cidr string) bool {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil || !ip.Equal(ipnet.IP) {
		return false
	}
	return true
}

func GetFamily(cidr string) (IPFamily, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, err
	}
	if ip.To4() == nil { // ipv6
		return IPv6, nil
	}
	return IPv4, nil
}

func OverlappingPrefixes(cidr1 string, cidr2 string) bool {
	_, ipnet1, err := net.ParseCIDR(cidr1)
	if err != nil {
		return false
	}
	_, ipnet2, err := net.ParseCIDR(cidr2)
	if err != nil {
		return false
	}
	return ipnet2.Contains(ipnet1.IP) || ipnet1.Contains(ipnet2.IP)
}

func NextPrefix(ipNet *net.IPNet) *net.IPNet {
	next := make([]byte, len(ipNet.IP))
	copy(next, ipNet.IP)
	maskLength, ipLength := ipNet.Mask.Size()
	wildcardLength := ipLength - maskLength
	var toAdd byte = 0
	var currentBlock int = len(ipNet.IP) - 1
	var carry byte = 1
	for i := 1; i <= ipLength; i++ {
		if i <= wildcardLength {
			toAdd = toAdd*2 + 1
		}
		if i%8 == 0 {
			if carry == 0 && toAdd == 0 {
				break
			}
			previousCarry := carry
			carry = 0
			if (int(next[currentBlock]) + int(toAdd) + int(previousCarry) + 1) > 255 {
				carry = 1
			}
			next[currentBlock] += toAdd + previousCarry
			currentBlock--
			toAdd = 0
		}
	}
	new := &net.IPNet{IP: next, Mask: ipNet.Mask}
	return new
}

func LastIP(ipNet *net.IPNet) net.IP {
	last := make([]byte, len(ipNet.IP))
	maskLength, ipLength := ipNet.Mask.Size()
	wildcardLength := ipLength - maskLength
	var toAdd byte = 0
	var currentBlock int = len(ipNet.IP) - 1
	for i := 1; i <= ipLength; i++ {
		if i <= wildcardLength {
			toAdd = toAdd*2 + 1
		}
		if i%8 == 0 {
			last[currentBlock] = ipNet.IP[currentBlock]
			last[currentBlock] += toAdd
			currentBlock--
			toAdd = 0
		}
	}
	return last
}
