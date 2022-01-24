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

package v1

import "fmt"

func (p *Prefix) ToString() string {
	return fmt.Sprintf("%s/%d", p.Address, p.PrefixLength)
}

// func (p *Prefixes) ToSlice() []string {
// 	res := []string{}
// 	for _, prefix := range p.Prefixes {
// 		res = append(res, prefix.ToString())
// 	}
// 	return res
// }

func (s *Subnet) ToString() string {
	return fmt.Sprintf("%s.%s.%s", s.Conduit.GetName(), s.Conduit.GetTrench().GetName(), s.Node)
}
