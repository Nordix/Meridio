/*
Copyright (c) 2022 Nordix Foundation

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

package nat

import (
	"fmt"

	"github.com/google/nftables"
	"github.com/sirupsen/logrus"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

/*
nft add table inet meridio-nat
nft add chain inet meridio-nat tcp-80-8080 {type nat hook prerouting priority 100\;}
nft flush chain inet meridio-nat tcp-80-8080
nft add set inet meridio-nat tcp-80-8080-ipv4 { type ipv4_addr\; }
nft add set inet meridio-nat tcp-80-8080-ipv6 { type ipv6_addr\; }
nft --debug all add rule inet meridio-nat tcp-80-8080 ip daddr @tcp-80-8080-ipv4 tcp dport 80 tcp dport set 8080 counter notrack
nft --debug all add rule inet meridio-nat tcp-80-8080 ip saddr @tcp-80-8080-ipv4 tcp sport 8080 tcp sport set 80 counter notrack
nft --debug all add rule inet meridio-nat tcp-80-8080 ip6 daddr @tcp-80-8080-ipv6 tcp dport 80 tcp dport set 8080 counter notrack
nft --debug all add rule inet meridio-nat tcp-80-8080 ip6 saddr @tcp-80-8080-ipv6 tcp sport 8080 tcp sport set 80 counter notrack
*/

const (
	tableName = "meridio-nat"
)

type NatHandler struct {
	table              *nftables.Table
	destinationPortNat map[string]*DestinationPortNat
}

func NewNatHandler() (*NatHandler, error) {
	nh := &NatHandler{
		destinationPortNat: map[string]*DestinationPortNat{},
	}
	err := nh.initTable()
	if err != nil {
		return nil, err
	}
	return nh, nil
}

func (nh *NatHandler) initTable() error {
	conn := &nftables.Conn{}
	nh.table = conn.AddTable(&nftables.Table{
		Name:   tableName,
		Family: nftables.TableFamilyINet,
	})
	return conn.Flush()
}

func (nh *NatHandler) SetNats(portNats []*nspAPI.Conduit_PortNat) error {
	var errFinal error
	toRemove := nspAPI.PortNatDiff(nh.getPortNats(), portNats)
	logrus.WithFields(logrus.Fields{
		"Previous Port Nats": nh.getPortNats(),
		"To add/update":      portNats,
		"To remove":          toRemove,
	}).Infof("NAT Handler: SetNats")
	for _, n := range toRemove {
		err := nh.deleteNat(n)
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
		}
	}
	for _, n := range portNats {
		err := nh.setNat(n)
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
		}
	}
	return errFinal
}

func (nh *NatHandler) setNat(portNat *nspAPI.Conduit_PortNat) error {
	portNatName := portNat.GetNatName()
	nat, exists := nh.destinationPortNat[portNatName]
	if exists { // update
		return nat.SetVips(portNat.GetVips())
	}
	// add
	nat, err := NewDestinationPortNat(nh.table, portNat)
	if err != nil {
		return err
	}
	nh.destinationPortNat[portNatName] = nat
	return nil
}

func (nh *NatHandler) deleteNat(portNat *nspAPI.Conduit_PortNat) error {
	portNatName := portNat.GetNatName()
	nat, exists := nh.destinationPortNat[portNatName]
	if !exists {
		return nil
	}
	delete(nh.destinationPortNat, portNatName)
	return nat.Delete()
}

func (nh *NatHandler) getPortNats() []*nspAPI.Conduit_PortNat {
	portNats := []*nspAPI.Conduit_PortNat{}
	for _, n := range nh.destinationPortNat {
		portNats = append(portNats, n.PortNat)
	}
	return portNats
}
