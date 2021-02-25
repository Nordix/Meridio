package networking

import (
	"reflect"

	"github.com/vishvananda/netlink"
)

const (
	NSE = iota // Interface linked to a NSC (e.g. target)
	NSC        // Interface linked to a NSE (e.g. Load balancer)
)

type InteraceType int

type Interface struct {
	index        int
	LocalIPs     []*netlink.Addr
	NeighborIPs  []*netlink.Addr
	InteraceType InteraceType
}

func (intf *Interface) getLink() (netlink.Link, error) {
	return netlink.LinkByIndex(intf.index)
}

func (intf *Interface) GetIndex() int {
	return intf.index
}

func (intf *Interface) GetName() string {
	i, err := intf.getLink()
	if err != nil {
		return ""
	}
	return i.Attrs().Name
}

func (intf *Interface) AddAddress(address *netlink.Addr) error {
	address.Label = ""
	i, err := intf.getLink()
	if err != nil {
		return err
	}
	err = netlink.AddrAdd(i, address)
	if err != nil {
		return err
	}
	return nil
}

func (intf *Interface) RemoveAddress(address *netlink.Addr) error {
	i, err := intf.getLink()
	if err != nil {
		return err
	}
	err = netlink.AddrDel(i, address)
	if err != nil {
		return err
	}
	return nil
}

// Get the index of an interface from its name
func (intf *Interface) Equals(intf2 *Interface) bool {
	return reflect.DeepEqual(intf, intf2)
}

// GetIndexFromName - Get the index of an interface from its name
func GetIndexFromName(name string) (int, error) {
	intf, err := netlink.LinkByName(name)
	if err != nil {
		return -1, err
	}
	return intf.Attrs().Index, nil
}

func NewInterface(index int, localIPs []*netlink.Addr, neighborIPs []*netlink.Addr) *Interface {
	intf := &Interface{
		index:        index,
		LocalIPs:     localIPs,
		NeighborIPs:  neighborIPs,
		InteraceType: -1,
	}
	return intf
}
