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

package kernel

import "github.com/vishvananda/netlink"

func AddVIP(vip string) error {
	netlinkAddr, err := netlink.ParseAddr(vip)
	if err != nil {
		return err
	}
	loopbackInterface, err := netlink.LinkByName("lo")
	if err != nil {
		return err
	}
	err = netlink.AddrAdd(loopbackInterface, netlinkAddr)
	if err != nil {
		return err
	}
	return nil
}

func DeleteVIP(vip string) error {
	netlinkAddr, err := netlink.ParseAddr(vip)
	if err != nil {
		return err
	}
	loopbackInterface, err := netlink.LinkByName("lo")
	if err != nil {
		return err
	}
	err = netlink.AddrDel(loopbackInterface, netlinkAddr)
	if err != nil {
		return err
	}
	return nil
}
