package networking

import (
	"fmt"
	"os/exec"
	"strconv"

	"github.com/vishvananda/netlink"
)

type FWMarkRoute struct {
	ip      *netlink.Addr
	fwmark  int
	tableId int
}

func (fwmr *FWMarkRoute) Delete() error {
	// TODO
	return nil
}

func (fwmr *FWMarkRoute) configure() error {
	cmd := "/usr/sbin/ip rule add fwmark " + strconv.Itoa(fwmr.fwmark) + " table " + strconv.Itoa(fwmr.tableId)
	_, err := exec.Command("bash", "-c", cmd).Output()

	if err != nil {
		return fmt.Errorf("%w; FWMarkRoute configure: %v", err, cmd)
	}

	cmd = "/usr/sbin/ip route add default via " + fwmr.ip.IP.String() + " table " + strconv.Itoa(fwmr.tableId)
	_, err = exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Errorf("%w; FWMarkRoute configure: %v", err, cmd)
	}

	return nil
}

func NewFWMarkRoute(ip *netlink.Addr, fwmark int, tableId int) (*FWMarkRoute, error) {
	fwMarkRoute := &FWMarkRoute{
		ip:      ip,
		fwmark:  fwmark,
		tableId: tableId,
	}
	err := fwMarkRoute.configure()
	if err != nil {
		return nil, err
	}
	return fwMarkRoute, nil
}
