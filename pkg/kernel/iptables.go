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
