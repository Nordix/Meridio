package networking

import (
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type Bridge struct {
	name             string
	linkedInterfaces []*Interface
	Interface
}

func (b *Bridge) create() error {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: b.name,
		},
	}
	err := netlink.LinkAdd(bridge)
	if err != nil {
		return err
	}
	err = netlink.LinkSetUp(bridge)
	if err != nil {
		return err
	}
	b.index = bridge.Index
	logrus.Infof("Bridge: Creation (id : %v), (name: %v)", b.index, b.name)
	return nil
}

func (b *Bridge) useExistingBridge() error {
	index, err := GetIndexFromName(b.name)
	if err != nil {
		return err
	}
	b.index = index
	logrus.Infof("Bridge: Use existing (id : %v), (name: %v)", b.index, b.name)
	return nil
}

func (b *Bridge) Delete() error {
	return nil
}

func (b *Bridge) addTolinkedInterfaces(intf *Interface) {
	b.linkedInterfaces = append(b.linkedInterfaces, intf)
}

func (b *Bridge) removeFromlinkedInterfaces(intf *Interface) {
	for index, i := range b.linkedInterfaces {
		if i.Equals(intf) {
			b.linkedInterfaces = append(b.linkedInterfaces[:index], b.linkedInterfaces[index+1:]...)
		}
	}
}

func (b *Bridge) interfaceIsLinked(intf *Interface) bool {
	for _, i := range b.linkedInterfaces {
		if i.Equals(intf) {
			return true
		}
	}
	return false
}

// LinkInterface set the bridge as master of another interface
func (b *Bridge) LinkInterface(intf *Interface) error {
	if b.interfaceIsLinked(intf) {
		return nil // TODO
	}
	b.addTolinkedInterfaces(intf)
	bridgeLink, err := b.getLink()
	if err != nil {
		return err
	}
	interfaceLink, err := intf.getLink()
	if err != nil {
		return err
	}
	err = netlink.LinkSetMaster(interfaceLink, bridgeLink)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bridge) UnLinkInterface(intf *Interface) error {
	if b.interfaceIsLinked(intf) == false {
		return nil // TODO
	}
	b.removeFromlinkedInterfaces(intf)
	interfaceLink, err := intf.getLink()
	if err != nil {
		return err
	}
	return netlink.LinkSetNoMaster(interfaceLink)
}

func NewBridge(name string) (*Bridge, error) {
	bridge := &Bridge{
		name:             name,
		linkedInterfaces: []*Interface{},
		Interface:        Interface{},
	}
	err := bridge.create()
	if err != nil {
		err = bridge.useExistingBridge()
		if err != nil {
			return nil, err
		}
	}
	return bridge, nil
}
