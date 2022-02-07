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

func TrenchFilter(filter *Trench, comparedTo *Trench) bool {
	if filter == nil && comparedTo != nil {
		return true
	}
	if comparedTo == nil {
		return false
	}
	name := filter.GetName() == comparedTo.GetName()
	if filter.GetName() == "" {
		name = true
	}
	return name
}

func ConduitFilter(filter *Conduit, comparedTo *Conduit) bool {
	if filter == nil && comparedTo != nil {
		return true
	}
	if comparedTo == nil {
		return false
	}
	name := filter.GetName() == comparedTo.GetName()
	if filter.GetName() == "" {
		name = true
	}
	parent := TrenchFilter(filter.GetTrench(), comparedTo.GetTrench())
	if filter.GetTrench() == nil {
		parent = true
	}
	return name && parent
}

func StreamFilter(filter *Stream, comparedTo *Stream) bool {
	if filter == nil && comparedTo != nil {
		return true
	}
	if comparedTo == nil {
		return false
	}
	name := filter.GetName() == comparedTo.GetName()
	if filter.GetName() == "" {
		name = true
	}
	parent := ConduitFilter(filter.GetConduit(), comparedTo.GetConduit())
	if filter.GetConduit() == nil {
		parent = true
	}
	return name && parent
}
