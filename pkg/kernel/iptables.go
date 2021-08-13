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
	"os/exec"
	"strconv"

	"github.com/vishvananda/netlink"
)

type NFQueue struct {
	ip       *netlink.Addr
	queueNum int
}

func (nfq *NFQueue) Delete() error {
	iptables := nfq.iptables()
	ipTablesCmd := exec.Command(iptables,
		"-t",
		"mangle",
		"-D",
		"PREROUTING",
		"-d",
		nfq.ip.String(),
		"-j",
		"NFQUEUE",
		"--queue-num",
		strconv.Itoa(nfq.queueNum))
	_, err := ipTablesCmd.Output()
	return err
}

func (nfq *NFQueue) configure() error {
	iptables := nfq.iptables()
	ipTablesCmd := exec.Command(iptables,
		"-t",
		"mangle",
		"-A",
		"PREROUTING",
		"-d",
		nfq.ip.String(),
		"-j",
		"NFQUEUE",
		"--queue-num",
		strconv.Itoa(nfq.queueNum))
	_, err := ipTablesCmd.Output()
	return err
}

func (nfq *NFQueue) iptables() string {
	if nfq.ip.IP.To4() != nil {
		return "iptables"
	}
	return "ip6tables"
}

func NewNFQueue(ip string, queueNum int) (*NFQueue, error) {
	netlinkAddr, err := netlink.ParseAddr(ip)
	if err != nil {
		return nil, err
	}
	nfQueue := &NFQueue{
		ip:       netlinkAddr,
		queueNum: queueNum,
	}
	err = nfQueue.configure()
	if err != nil {
		return nil, err
	}
	return nfQueue, nil
}
