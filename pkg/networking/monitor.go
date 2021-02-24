package networking

import (
	"syscall"

	"github.com/vishvananda/netlink"
)

type InterfaceMonitor struct {
	ch          chan netlink.LinkUpdate
	done        chan struct{}
	flush       chan struct{}
	subscribers []InterfaceMonitorSubscriber
}

type InterfaceMonitorSubscriber interface {
	InterfaceCreated(*Interface)
	InterfaceDeleted(*Interface)
}

// Subscribe -
func (im *InterfaceMonitor) Subscribe(subscriber InterfaceMonitorSubscriber) {
	im.subscribers = append(im.subscribers, subscriber)
}

// UnSubscribe -
func (im *InterfaceMonitor) UnSubscribe(subscriber InterfaceMonitorSubscriber) {
	for index, current := range im.subscribers {
		if subscriber == current {
			im.subscribers = append(im.subscribers[:index], im.subscribers[index+1:]...)
		}
	}
}

func (im *InterfaceMonitor) interfaceCreated(link netlink.Link) {
	for _, subscriber := range im.subscribers {
		intf := NewInterface(link.Attrs().Index, []*netlink.Addr{}, []*netlink.Addr{})
		subscriber.InterfaceCreated(intf)
	}
}

func (im *InterfaceMonitor) interfaceDeleted(link netlink.Link) {
	for _, subscriber := range im.subscribers {
		intf := NewInterface(link.Attrs().Index, []*netlink.Addr{}, []*netlink.Addr{})
		subscriber.InterfaceDeleted(intf)
	}
}

// Start -
func (im *InterfaceMonitor) start() {
	for {
		select {
		case update, ok := <-im.ch:
			if !ok {
				im.Close()
				return
			}
			switch update.Header.Type {
			case syscall.RTM_NEWLINK:
				im.interfaceCreated(update.Link)
			case syscall.RTM_DELLINK:
				im.interfaceDeleted(update.Link)
			}
		case _ = <-im.flush:
			continue
		}
	}
}

func (im *InterfaceMonitor) eventSubscription() error {
	err := netlink.LinkSubscribe(im.ch, im.done)
	if err != nil {
		return err
	}
	return nil
}

// Close -
func (im *InterfaceMonitor) Close() {
	close(im.done)
}

// NewLinkMonitor -
func NewLinkMonitor() (*InterfaceMonitor, error) {
	interfaceMonitor := &InterfaceMonitor{
		ch:    make(chan netlink.LinkUpdate),
		done:  make(chan struct{}),
		flush: make(chan struct{}),
	}

	err := interfaceMonitor.eventSubscription()
	if err != nil {
		return nil, err
	}
	go interfaceMonitor.start()

	return interfaceMonitor, nil
}
