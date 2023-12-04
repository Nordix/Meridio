/*
Copyright (c) 2021-2023 Nordix Foundation

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
	"context"
	"strings"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

	"github.com/nordix/meridio/pkg/log"
)

const MAX_INTERFACE_NAME_LENGTH = 15

type interfaceNameSetter struct {
	nameCache *InterfaceNameChache
	prefix    string
	maxLength int
}

func (ins *interfaceNameSetter) SetInterfaceName(request *networkservice.NetworkServiceRequest) {
	ins.setInterfaceNameMechanism(request)
}

func (ins *interfaceNameSetter) UnsetInterfaceName(obj interface{}) {
	releaseFunc := func(mechanism *networkservice.Mechanism, id string) {
		_, exists := mechanism.GetParameters()[common.InterfaceNameKey]
		if exists {
			ins.nameCache.Release(id)
		}
	}

	switch obj := obj.(type) {
	case *networkservice.Connection: // to be used when closing established connection via NSM Close()
		conn := obj
		if conn == nil || conn.GetMechanism() == nil || conn.GetMechanism().GetParameters() == nil {
			return
		}
		releaseFunc(conn.GetMechanism(), conn.GetId())
	case *networkservice.NetworkServiceRequest: // to be used when NSM Request() fails
		// Release interface name when a connection couldn't be established so that it
		// won't be leaked (e.g. when client user gives up).
		// But do not return the interface name in case of failed refresh/update attempts
		// on established connections. That should be taken care of by NSM Close().
		request := obj
		if request == nil || request.GetMechanismPreferences() == nil || request.GetConnection() == nil {
			return
		}
		if request.GetConnection().GetMechanism() != nil {
			// established connection (I hope non-nil Mechanism is equvivalent to that)
			return
		}
		for _, mechanism := range request.GetMechanismPreferences() {
			if mechanism.GetParameters() == nil {
				continue
			}
			releaseFunc(mechanism, request.GetConnection().GetId())
		}
	}
}

func (ins *interfaceNameSetter) setInterfaceNameMechanism(request *networkservice.NetworkServiceRequest) {
	if request == nil || request.GetConnection() == nil || request.GetConnection().GetMechanism() == nil {
		ins.setInterfaceNameMechanismPreferences(request)
		return
	}
	mechanism := request.GetConnection().GetMechanism()
	if mechanism.GetParameters() == nil {
		mechanism.Parameters = make(map[string]string)
	}
	// Do not generate new local interface name when Request for an established connection
	// is resent by the refresh chain component.
	// Also, if the name is set but does not match the prefix overwrite it.
	id := request.GetConnection().GetId()
	if val, ok := mechanism.GetParameters()[common.InterfaceNameKey]; !ok ||
		val == "" || (ins.prefix != "" && !strings.HasPrefix(val, ins.prefix)) {
		// XXX: In theory if val exists but gets overwritten, then old value might need to be released.
		// But considering that currently only format issues would trigger overwrite, there's no (?)
		// chance for this, as MechanismPreferences had to be vetted before.
		var interfaceName string
		if val == "" {
			// TAPA can clear the interface name from the connection to indicate connection
			// was restored via connection monitor, thus cache update might be necessary.
			// In such cases, TAPA passes the removed interface name in Mechanism Preferences.
			// So, check if Mechanism Preferences contain a suggested interface name (matching the prefix).
			// Then check if the name could be used (not in use by some other connection, or
			// there's no other name associated with the connection ID according to the cache).
			// If Mechanism Preferences does not contain a feasible interface name, we should
			// generate a new one. Cache must reflect the outcome at the end.
			for _, prefMechanism := range request.GetMechanismPreferences() {
				if prefMechanism.Cls != mechanism.Cls ||
					prefMechanism.Type != mechanism.Type ||
					prefMechanism.GetParameters() == nil {
					continue
				}
				prefVal, ok := prefMechanism.GetParameters()[common.InterfaceNameKey]
				if !ok || len(prefVal) > ins.maxLength || (ins.prefix != "" && !strings.HasPrefix(prefVal, ins.prefix)) {
					continue
				}
				// Consult the cache if the "preferred" interface name could be used, and reserve it.
				// If threre's already a cached value we shall use that instead.
				if interfaceName = ins.nameCache.CheckAndReserve(id, prefVal, ins.prefix, ins.maxLength); interfaceName != "" {
					// Use the returned interface name for the connection
					// TODO: If the request fails to establish connection, the inteface name will not be released
					log.Logger.Info("setInterfaceNameMechanism", "connection ID", id, "interface", interfaceName, "preferred interface", prefVal)
					break
				}
			}

		}
		if interfaceName == "" {
			interfaceName = ins.nameCache.Generate(id, ins.prefix, ins.maxLength)
		}
		mechanism.GetParameters()[common.InterfaceNameKey] = interfaceName
	}
}

func (ins *interfaceNameSetter) setInterfaceNameMechanismPreferences(request *networkservice.NetworkServiceRequest) {
	if request == nil || request.GetMechanismPreferences() == nil {
		return
	}
	for _, mechanism := range request.GetMechanismPreferences() {
		if mechanism.GetParameters() == nil {
			mechanism.Parameters = map[string]string{}
		}
		// Overwrite if the name does not match the expected format (prefix mismatch).
		// Do not generate new local interface name when Request is resent with a valid interface name.
		// Although, double checking the inteface name cache might be a good idea even if the format is ok.
		//
		// Note: NSM's kernelMechanismClient will add interface name to mechanism preferences if not present.
		// Thus it's crucial that the generated name does not match the prefix!!! Meaning, that the Network
		// Service name must not start with the prefix.
		//
		// TODO: How to handle interface names in MechanismPreferences that match the prefix? One of them might
		// get accepted. If the connection gets established that interface name won't be stored in the cache.
		// We should either forbid passing "valid" preferred interface names or do sg about the cache.
		if val, ok := mechanism.Parameters[common.InterfaceNameKey]; !ok ||
			val == "" || (ins.prefix != "" && !strings.HasPrefix(val, ins.prefix)) {
			// Note: If the request gets cancelled before the connection is established or simply the request
			// fails, the inteface name has to be released.
			// Note: In case of multiple MechanismPreferences only one can get accepted as the interface name
			// of the connection. Luckily cache allows only 1 name per connection ID. So we shall not end up with
			// leaked names.
			interfaceName := ins.nameCache.Generate(request.GetConnection().GetId(), ins.prefix, ins.maxLength)
			mechanism.Parameters[common.InterfaceNameKey] = interfaceName
		}
	}
}

// NewInterfaceNameEndpoint -
func newInterfaceNameSetter(prefix string, generator NameGenerator, maxLength int) *interfaceNameSetter {
	return &interfaceNameSetter{
		nameCache: NewInterfaceNameChache(context.TODO(), generator),
		prefix:    prefix,
		maxLength: maxLength,
	}
}
