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
