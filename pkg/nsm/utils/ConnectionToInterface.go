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

package nsm

import (
	"fmt"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/networking"
)

func ConvertConnectionToInterface(interfaceName string, conn *networkservice.Connection, interfaceType networking.InterfaceType) (networking.Iface, error) {

	// netUtils networking.Utils
	netUtils := &kernel.KernelUtils{}

	index, err := netUtils.GetIndexFromName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("interface with name %s does not exist", interfaceName)
	}

	intf := netUtils.NewInterface(index)
	intf.SetInterfaceType(interfaceType)

	ConnectionContext := conn.GetContext()
	if ConnectionContext == nil {
		return intf, nil
	}

	IpContext := ConnectionContext.GetIpContext()
	if IpContext == nil {
		return intf, nil
	}

	intf.SetLocalPrefixes(IpContext.DstIpAddrs)
	intf.SetNeighborPrefixes(IpContext.DstIpAddrs)
	intf.SetGatewayPrefixes(IpContext.ExtraPrefixes)

	return intf, nil
}
