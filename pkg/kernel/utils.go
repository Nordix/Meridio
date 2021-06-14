package kernel

import (
	"github.com/nordix/meridio/pkg/networking"
	"github.com/vishvananda/netlink"
)

type KernelUtils struct {
}

func (ku *KernelUtils) NewInterface(index int) networking.Iface {
	return NewInterface(index)
}

func (ku *KernelUtils) NewBridge(name string) (networking.Bridge, error) {
	return NewBridge(name)
}

func (ku *KernelUtils) NewFWMarkRoute(ip string, fwmark int, tableID int) (networking.FWMarkRoute, error) {
	return NewFWMarkRoute(ip, fwmark, tableID)
}

func (ku *KernelUtils) NewNFQueue(prefix string, queueNum int) (networking.NFQueue, error) {
	return NewNFQueue(prefix, queueNum)
}

func (ku *KernelUtils) NewSourceBasedRoute(tableID int, prefix string) (networking.SourceBasedRoute, error) {
	return NewSourceBasedRoute(tableID, prefix)
}

func (ku *KernelUtils) NewInterfaceMonitor() (networking.InterfaceMonitor, error) {
	return NewInterfaceMonitor()
}

// func (ku *KernelUtils) IfaceByName(name string) (networking.Iface, error) {

// }

func (ku *KernelUtils) GetIndexFromName(name string) (int, error) {
	return GetIndexFromName(name)
}

func (ku *KernelUtils) AddVIP(vip string) error {
	return AddVIP(vip)
}

func (ku *KernelUtils) DeleteVIP(vip string) error {
	return DeleteVIP(vip)
}

func getLink(intf networking.Iface) (netlink.Link, error) {
	return netlink.LinkByIndex(intf.GetIndex())
}

// GetIndexFromName - Get the index of an interface from its name
func GetIndexFromName(name string) (int, error) {
	intf, err := netlink.LinkByName(name)
	if err != nil {
		return -1, err
	}
	return intf.Attrs().Index, nil
}
