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

package v1

import "fmt"

// Checks the differences between 2 port nat list
// 1st return gives the items added
// 2nd return gives the items in common
// 3rd return gives the items removed
func PortNatDiff(old []*Conduit_PortNat, new []*Conduit_PortNat) ([]*Conduit_PortNat, []*Conduit_PortNat, []*Conduit_PortNat) {
	added := []*Conduit_PortNat{}
	common := []*Conduit_PortNat{}
	removed := []*Conduit_PortNat{}
	oldMap := map[string]*Conduit_PortNat{}
	for _, o := range old {
		oldMap[o.GetNatName()] = o
	}
	for _, n := range new {
		_, exists := oldMap[n.GetNatName()]
		if exists {
			common = append(common, n)
			delete(oldMap, n.GetNatName())
		} else {
			added = append(added, n)
		}
	}
	for _, portNat := range oldMap {
		removed = append(removed, portNat)
	}
	return added, common, removed
}

func (pn *Conduit_PortNat) GetNatName() string {
	return fmt.Sprintf("%d-%d-%s", pn.Port, pn.TargetPort, pn.Protocol)
}
