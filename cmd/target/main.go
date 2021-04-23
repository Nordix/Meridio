package main

import (
	"context"
	"hash/fnv"
	"os"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nordix/meridio/pkg/client"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	var config Config
	err := envconfig.Process("nsm", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	_, err = netlink.ParseAddr(config.VIP)
	if err != nil {
		logrus.Fatalf("Error Parsing the VIP: %v", err)
	}

	netUtils := &linuxKernel.KernelUtils{}

	// nspClient, _ := nsp.NewNetworkServicePlateformClient(config.NSPService)
	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	// st := &SimpleTarget{
	// 	networkServicePlateformClient: nspClient,
	// 	identifier:                    identifier,
	// }
	st := NewSimpleTarget(identifier, config.VIP, netUtils)

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	apiClient := nsm.NewAPIClient(ctx, apiClientConfig)

	extraContext := make(map[string]string)
	extraContext["identifier"] = strconv.Itoa(st.identifier)

	client := client.NewNetworkServiceClient(config.ProxyNetworkServiceName, apiClient)
	client.InterfaceMonitorSubscriber = st
	client.ExtraContext = extraContext
	client.Request()

	for {
		time.Sleep(10 * time.Second)
	}
}

// Hash -
func Hash(s string, n int) int {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		logrus.Fatalf("error hashing: %+v", err)
	}
	return int(h.Sum32())%n + 1
}

// SimpleTarget -
type SimpleTarget struct {
	identifier       int
	sourceBasedRoute networking.SourceBasedRoute
	vip              string
}

// InterfaceCreated -
func (st *SimpleTarget) InterfaceCreated(intf networking.Iface) {
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("SimpleTarget: Adding nexthop: no gateway: %v", intf)
		return
	}
	gateway := intf.GetGatewayPrefixes()[0]
	err := st.sourceBasedRoute.AddNexthop(gateway)
	if err != nil {
		logrus.Errorf("SimpleTarget: Adding nexthop (%v) to source base route err: %v", gateway, err)
	}
}

// InterfaceDeleted -
func (st *SimpleTarget) InterfaceDeleted(intf networking.Iface) {
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("SimpleTarget: Removing nexthop: no gateway: %v", intf)
		return
	}
	gateway := intf.GetGatewayPrefixes()[0]
	err := st.sourceBasedRoute.RemoveNexthop(gateway)
	if err != nil {
		logrus.Errorf("SimpleTarget: Removing nexthop (%v) from source base route err: %v", gateway, err)
	}
}

func NewSimpleTarget(identifier int, vip string, netUtils networking.Utils) *SimpleTarget {
	sourceBasedRoute, err := netUtils.NewSourceBasedRoute(10, vip)
	if err != nil {
		logrus.Errorf("SimpleTarget: NewSourceBasedRoute err: %v", err)
	}
	err = netUtils.AddVIP(vip)
	if err != nil {
		logrus.Errorf("SimpleTarget: err AddVIP: %v", err)
	}
	simpleTarget := &SimpleTarget{
		identifier:       identifier,
		sourceBasedRoute: sourceBasedRoute,
		vip:              vip,
	}
	return simpleTarget
}
