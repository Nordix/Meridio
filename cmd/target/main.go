package main

import (
	"context"
	"hash/fnv"
	"os"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsp"
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

	vip, err := netlink.ParseAddr(config.VIP)
	if err != nil {
		logrus.Fatalf("Error Parsing the VIP: %v", err)
	}

	// todo
	// temporary solution
	// Wait for the proxies to be created
	time.Sleep(35 * time.Second)

	nspClient, _ := nsp.NewNetworkServicePlateformClient(config.NSPService)
	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	// st := &SimpleTarget{
	// 	networkServicePlateformClient: nspClient,
	// 	identifier:                    identifier,
	// }
	st := NewSimpleTarget(nspClient, identifier, vip)

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	apiClient := nsm.NewAPIClient(ctx, apiClientConfig)

	client := client.NewNetworkServiceClient(config.ProxyNetworkServiceName, apiClient)
	client.InterfaceMonitorSubscriber = st
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
	networkServicePlateformClient *nsp.NetworkServicePlateformClient
	identifier                    int
	sourceBasedRoute              *networking.SourceBasedRoute
	vip                           *netlink.Addr
}

// InterfaceCreated -
func (st *SimpleTarget) InterfaceCreated(intf *networking.Interface) {
	context := make(map[string]string)
	context["identifier"] = strconv.Itoa(st.identifier)
	err := st.networkServicePlateformClient.Register(intf.LocalIPs[0].String(), context)
	if err != nil {
		logrus.Errorf("SimpleTarget: Register err: %v", err)
	}
	err = st.sourceBasedRoute.AddNexthop(intf.NeighborIPs[0])
	if err != nil {
		logrus.Errorf("SimpleTarget: Adding nexthop (%v) to source base route err: %v", intf.NeighborIPs[0], err)
	}
}

// InterfaceDeleted -
func (st *SimpleTarget) InterfaceDeleted(intf *networking.Interface) {
	err := st.networkServicePlateformClient.Unregister(intf.LocalIPs[0].String())
	if err != nil {
		logrus.Errorf("SimpleTarget: Unregister err: %v", err)
	}
}

func NewSimpleTarget(networkServicePlateformClient *nsp.NetworkServicePlateformClient, identifier int, vip *netlink.Addr) *SimpleTarget {
	sourceBasedRoute, err := networking.NewSourceBasedRoute(10, vip)
	if err != nil {
		logrus.Errorf("SimpleTarget: NewSourceBasedRoute err: %v", err)
	}
	err = networking.AddVIP(vip)
	if err != nil {
		logrus.Errorf("SimpleTarget: err AddVIP: %v", err)
	}
	simpleTarget := &SimpleTarget{
		networkServicePlateformClient: networkServicePlateformClient,
		identifier:                    identifier,
		sourceBasedRoute:              sourceBasedRoute,
		vip:                           vip,
	}
	return simpleTarget
}
