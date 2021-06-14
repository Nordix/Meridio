package loadbalancer

import (
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
)

type virtualIP struct {
	prefix   string
	netUtils networking.Utils
	nfqueue  networking.NFQueue
}

func (vip *virtualIP) Delete() error {
	return vip.nfqueue.Delete()
}

func (vip *virtualIP) createNFQueue() error {
	var err error
	vip.nfqueue, err = vip.netUtils.NewNFQueue(vip.prefix, 2)
	if err != nil {
		logrus.Errorf("Load Balancer: error configuring nfqueue (iptables): %v", err)
		return err
	}
	return nil
}

func newVirtualIP(prefix string, netUtils networking.Utils) (*virtualIP, error) {
	vip := &virtualIP{
		prefix:   prefix,
		netUtils: netUtils,
	}
	err := vip.createNFQueue()
	if err != nil {
		return nil, err
	}
	return vip, nil
}
