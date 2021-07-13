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

package interfacename

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
)

const MAX_INTERFACE_NAME_LENGTH = 16

type interfaceNameSetter struct {
	nameGenerator NameGenerator
	prefix        string
	maxLength     int
}

func (ins *interfaceNameSetter) SetInterfaceName(request *networkservice.NetworkServiceRequest) {
	ins.setInterfaceNameMechanism(request)
	ins.setInterfaceNameMechanismPreferences(request)
}

func (ins *interfaceNameSetter) setInterfaceNameMechanism(request *networkservice.NetworkServiceRequest) {
	if request == nil || request.GetConnection() == nil || request.GetConnection().GetMechanism() == nil {
		return
	}
	mechanism := request.GetConnection().GetMechanism()
	if mechanism.GetParameters() == nil {
		mechanism.Parameters = make(map[string]string)
	}
	// Do not generate new local interface name when Request for an established connection
	// is resent by the refresh chain component.
	if val, ok := mechanism.GetParameters()[common.InterfaceNameKey]; !ok || val == "" {
		mechanism.GetParameters()[common.InterfaceNameKey] = ins.nameGenerator.Generate(ins.prefix, ins.maxLength)
	}
}

func (ins *interfaceNameSetter) setInterfaceNameMechanismPreferences(request *networkservice.NetworkServiceRequest) {
	if request == nil || request.GetMechanismPreferences() == nil {
		return
	}
	for _, mechanism := range request.GetMechanismPreferences() {
		if mechanism.Parameters == nil {
			mechanism.Parameters = map[string]string{}
		}
		// Do not generate new local interface name when Request for an established connection
		// is resent by the refresh chain component. (Does it even make sense to generate a new
		// interfae name for MechanismPreferences during connection refresh? (Interface in use is
		// present in Mechanism.))
		if val, ok := mechanism.Parameters[common.InterfaceNameKey]; !ok || val == "" {
			mechanism.Parameters[common.InterfaceNameKey] = ins.nameGenerator.Generate(ins.prefix, ins.maxLength)
		}
	}
}

// NewInterfaceNameEndpoint -
func newInterfaceNameSetter(prefix string, generator NameGenerator, maxLength int) *interfaceNameSetter {
	return &interfaceNameSetter{
		nameGenerator: generator,
		prefix:        prefix,
		maxLength:     maxLength,
	}
}
