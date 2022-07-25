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
func PortNatDiff(set1 []*Conduit_PortNat, set2 []*Conduit_PortNat) []*Conduit_PortNat {
	diff := []*Conduit_PortNat{}
	set2Map := map[string]*Conduit_PortNat{}
	for _, pn := range set2 {
		set2Map[pn.GetNatName()] = pn
	}
	for _, pn := range set1 {
		_, exists := set2Map[pn.GetNatName()]
		if !exists {
			diff = append(diff, pn)
		}
	}
	return diff
}

func (pn *Conduit_PortNat) GetNatName() string {
	return fmt.Sprintf("%d-%d-%s", pn.Port, pn.TargetPort, pn.Protocol)
}
