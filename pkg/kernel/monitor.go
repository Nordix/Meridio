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

package kernel

import (
	"syscall"

	"github.com/nordix/meridio/pkg/networking"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type InterfaceMonitor struct {
	ch          chan netlink.LinkUpdate
	done        chan struct{}
	flush       chan struct{}
	subscribers []networking.InterfaceMonitorSubscriber
}

// Subscribe -
func (im *InterfaceMonitor) Subscribe(subscriber networking.InterfaceMonitorSubscriber) {
	im.subscribers = append(im.subscribers, subscriber)
}

// UnSubscribe -
func (im *InterfaceMonitor) UnSubscribe(subscriber networking.InterfaceMonitorSubscriber) {
	for index, current := range im.subscribers {
		if subscriber == current {
			im.subscribers = append(im.subscribers[:index], im.subscribers[index+1:]...)
		}
	}
}

func (im *InterfaceMonitor) interfaceCreated(link netlink.Link) {
	for _, subscriber := range im.subscribers {
		intf := NewInterface(link.Attrs().Index)
		subscriber.InterfaceCreated(intf)
	}
}

func (im *InterfaceMonitor) interfaceDeleted(link netlink.Link) {
	for _, subscriber := range im.subscribers {
		intf := NewInterface(link.Attrs().Index)
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
				if update.Link.Attrs().Flags&unix.IFF_UP != 0 {
					im.interfaceCreated(update.Link)
				}
			case syscall.RTM_DELLINK:
				im.interfaceDeleted(update.Link)
			}
		case <-im.flush:
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

// NewInterfaceMonitor -
func NewInterfaceMonitor() (*InterfaceMonitor, error) {
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
