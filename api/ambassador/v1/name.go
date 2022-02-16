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

func (t *Trench) FullName() string {
	return t.GetName()
}

func (c *Conduit) FullName() string {
	return fmt.Sprintf("%s.%s", c.GetName(), c.GetTrench().FullName())
}

func (s *Stream) FullName() string {
	return fmt.Sprintf("%s.%s", s.GetName(), s.GetConduit().FullName())
}
