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

import (
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

func (t *Trench) ToNSP() *nspAPI.Trench {
	return &nspAPI.Trench{
		Name: t.GetName(),
	}
}

func (c *Conduit) ToNSP() *nspAPI.Conduit {
	return &nspAPI.Conduit{
		Name:   c.GetName(),
		Trench: c.GetTrench().ToNSP(),
	}
}

func (s *Stream) ToNSP() *nspAPI.Stream {
	return &nspAPI.Stream{
		Name:    s.GetName(),
		Conduit: s.GetConduit().ToNSP(),
	}
}
