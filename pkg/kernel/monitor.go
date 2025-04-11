/*
Copyright (c) 2021 Nordix Foundation
Copyright (c) 2025 OpenInfra Foundation Europe

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
	"fmt"
	"sync"
	"syscall"

	"github.com/nordix/meridio/pkg/networking"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type InterfaceMonitor struct {
	ch                 chan netlink.LinkUpdate
	done               chan struct{}
	flush              chan struct{}
	subscribers        []networking.InterfaceMonitorSubscriber
	subscribersCopy    []networking.InterfaceMonitorSubscriber // allows subscriber notifications to proceed without the need of holding the mutex
	subscribersChanged bool                                    // indicates change in subsribers list
	mu                 sync.Mutex
}

// Subscribe -
func (im *InterfaceMonitor) Subscribe(subscriber networking.InterfaceMonitorSubscriber) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.subscribers = append(im.subscribers, subscriber)
	im.subscribersChanged = true
}

// UnSubscribe -
func (im *InterfaceMonitor) UnSubscribe(subscriber networking.InterfaceMonitorSubscriber) {
	im.mu.Lock()
	defer im.mu.Unlock()
	for index, current := range im.subscribers {
		if subscriber == current {
			im.subscribers = append(im.subscribers[:index], im.subscribers[index+1:]...)
			im.subscribersChanged = true
		}
	}
}

// updateSubscribersCopy - updates copy of subscribers if there had been a change
// Note: It is assumed that changes due to Subscribe/UnSubscribe calls are rare,
// thus subscribersCopy approach can provide a resource efficient protection against
// subscriber induced deadlocks.
func (im *InterfaceMonitor) updateSubscribersCopy() {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.subscribersCopy == nil || im.subscribersChanged {
		if len(im.subscribers) > 0 {
			im.subscribersCopy = make([]networking.InterfaceMonitorSubscriber, len(im.subscribers))
			copy(im.subscribersCopy, im.subscribers)
		} else {
			im.subscribersCopy = []networking.InterfaceMonitorSubscriber{}
		}
		im.subscribersChanged = false
	}
}

// Note: Due to the usage of subscribersCopy approach for deadlock prevention,
// in the case of a prolonged subscriber notification other subscribers who might
// have already unsubscribed could still get a single delayed notification.
// If that's a problem slow subscribers are encouraged to implement custom solutions
// to avoid blocking interfaceMonitor notifications.
func (im *InterfaceMonitor) interfaceCreated(link netlink.Link) {
	im.updateSubscribersCopy()

	for _, subscriber := range im.subscribersCopy {
		intf := NewInterface(link.Attrs().Index)
		subscriber.InterfaceCreated(intf)
	}
}

func (im *InterfaceMonitor) interfaceDeleted(link netlink.Link) {
	im.updateSubscribersCopy()

	for _, subscriber := range im.subscribersCopy {
		intf := NewInterface(link.Attrs().Index, WithInterfaceName(link.Attrs().Name))
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
		return fmt.Errorf("failed subscribing to link events: %w", err)
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
