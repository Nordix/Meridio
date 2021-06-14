package target

import (
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
)

type virtualIP struct {
	sourceBasedRoute networking.SourceBasedRoute
	prefix           string
	netUtils         networking.Utils
}

func (vip *virtualIP) Delete() error {
	err := vip.netUtils.DeleteVIP(vip.prefix)
	if err != nil {
		return err
	}
	return vip.removeSourceBaseRoute()
}

func (vip *virtualIP) AddNexthop(ip string) error {
	return vip.sourceBasedRoute.AddNexthop(ip)
}

func (vip *virtualIP) RemoveNexthop(ip string) error {
	return vip.sourceBasedRoute.RemoveNexthop(ip)
}

func (vip *virtualIP) createSourceBaseRoute(tableID int) error {
	var err error
	vip.sourceBasedRoute, err = vip.netUtils.NewSourceBasedRoute(tableID, vip.prefix)
	logrus.Infof("VIP Simple target: sourceBasedRoute index - vip: %v - %v", tableID, vip.prefix)
	if err != nil {
		return err
	}
	return nil
}

func (vip *virtualIP) removeSourceBaseRoute() error {
	return vip.sourceBasedRoute.Delete()
}

func newVirtualIP(prefix string, tableID int, netUtils networking.Utils) (*virtualIP, error) {
	vip := &virtualIP{
		prefix:   prefix,
		netUtils: netUtils,
	}
	err := vip.createSourceBaseRoute(tableID)
	if err != nil {
		return nil, err
	}
	err = vip.netUtils.AddVIP(vip.prefix)
	if err != nil {
		return nil, err
	}
	return vip, nil
}
