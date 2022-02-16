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

func (t1 *Trench) Equals(t2 *Trench) bool {
	return t1.GetName() == t2.GetName()
}

func (c1 *Conduit) Equals(c2 *Conduit) bool {
	return c1.GetName() == c2.GetName() && c1.GetTrench().Equals(c2.GetTrench())
}

func (s1 *Stream) Equals(s2 *Stream) bool {
	return s1.GetName() == s2.GetName() && s1.GetConduit().Equals(s2.GetConduit())
}
