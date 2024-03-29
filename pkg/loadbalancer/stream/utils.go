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

package stream

import (
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
)

var interfaceNamePrefix string = "load-balancer"

// SetInterfaceNamePrefix -
// Derives the NSM v1.2-rc1 (same as v1.1.1) interface name prefix based on the Network Service name of the NSE.
//
// Note: By letting NSM name the NSE interface there's no need for a custom
// interfacename.NewServer chain component.
func SetInterfaceNamePrefix(ns string) {
	nsMaxLength := kernelmech.LinuxIfMaxLength - 5
	if len(ns) > nsMaxLength {
		ns = ns[:nsMaxLength]
	}

	interfaceNamePrefix = ns
}

// GetInterfaceNamePrefix -
func GetInterfaceNamePrefix() string {
	return interfaceNamePrefix
}
