package main

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	nspAPI "github.com/nordix/meridio/api/nsp"
	"github.com/nordix/meridio/pkg/endpoint"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/loadbalancer"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
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

	_, err = netlink.ParseAddr(config.VIP)
	if err != nil {
		logrus.Fatalf("Error Parsing the VIP: %v", err)
	}

	netUtils := &linuxKernel.KernelUtils{}

	nspClient, err := nsp.NewNetworkServicePlateformClient(config.NSPService)
	if err != nil {
		logrus.Errorf("NewNetworkServicePlateformClient: %v", err)
	}
	sns := NewSimpleNetworkService(config.VIP, nspClient, netUtils)

	linkMonitor, err := netUtils.NewLinkMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}
	interfaceMonitorEndpoint := interfacemonitor.NewServer(sns, netUtils)
	linkMonitor.Subscribe(interfaceMonitorEndpoint)

	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		interfacename.NewServer("nse", &interfacename.RandomGenerator{}),
		interfaceMonitorEndpoint,
		sendfd.NewServer(),
	}

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	nsmAPIClient := nsm.NewAPIClient(ctx, apiClientConfig)

	endpointConfig := &endpoint.Config{
		Name:             config.Name,
		ServiceName:      config.ServiceName,
		Labels:           config.Labels,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	ep, err := endpoint.NewEndpoint(ctx, endpointConfig, nsmAPIClient.NetworkServiceRegistryClient, nsmAPIClient.NetworkServiceEndpointRegistryClient)
	if err != nil {
		logrus.Fatalf("unable to create a new nse %+v", err)
	}

	err = ep.Start(responderEndpoint...)
	if err != nil {
		logrus.Fatalf("unable to start nse %+v", err)
	}

	defer ep.Delete()

	sns.Start()

	<-ctx.Done()
}

// SimpleNetworkService -
type SimpleNetworkService struct {
	loadbalancer                         *loadbalancer.LoadBalancer
	networkServicePlateformClient        *nsp.NetworkServicePlateformClient
	vip                                  string
	networkServicePlateformServiceStream nspAPI.NetworkServicePlateformService_MonitorClient
}

// Start -
func (sns *SimpleNetworkService) Start() {
	var err error
	sns.networkServicePlateformServiceStream, err = sns.networkServicePlateformClient.Monitor()
	if err != nil {
		logrus.Errorf("SimpleNetworkService: err Monitor: %v", err)
	}
	go sns.recv()
}

func (sns *SimpleNetworkService) recv() {
	for {
		target, err := sns.networkServicePlateformServiceStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Errorf("SimpleNetworkService: event err: %v", err)
			break
		}

		lbTarget, err := sns.parseLoadBalancerTarget(target)
		if err != nil {
			logrus.Errorf("SimpleNetworkService: parseLoadBalancerTarget err: %v", err)
			continue
		}

		if target.Status == nspAPI.Status_Register {
			err = sns.loadbalancer.AddTarget(lbTarget)
			logrus.Infof("SimpleNetworkService: A Add Target: %v", target)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: err A AddTarget (%v): %v", target, err)
				continue
			}
		} else if target.Status == nspAPI.Status_Unregister {
			err = sns.loadbalancer.RemoveTarget(lbTarget)
			logrus.Infof("SimpleNetworkService: Remove Target: %v", target)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: err RemoveTarget (%v): %v", target, err)
				continue
			}
		}
	}
}

func (sns *SimpleNetworkService) parseLoadBalancerTarget(target *nspAPI.Target) (*loadbalancer.Target, error) {
	identifierStr, exists := target.Context["identifier"]
	if !exists {
		logrus.Errorf("SimpleNetworkService: identifier does not exist: %v", target.Context)
		return nil, errors.New("Identifier does not exist")
	}
	identifier, err := strconv.Atoi(identifierStr)
	if err != nil {
		logrus.Errorf("SimpleNetworkService: cannot parse identifier (%v): %v", identifierStr, err)
		return nil, err
	}
	_, err = netlink.ParseAddr(target.Ip)
	if err != nil {
		logrus.Errorf("SimpleNetworkService: cannot parse IP (%v): %v", target.Ip, err)
		return nil, err
	}
	return loadbalancer.NewTarget(identifier, target.Ip), nil
}

// InterfaceCreated -
func (sns *SimpleNetworkService) InterfaceCreated(intf networking.Iface) {
	// todo
	logrus.Infof("SimpleNetworkService: InterfaceCreated: %v", intf)
	go func() {
		time.Sleep(2 * time.Second)
		targets, err := sns.networkServicePlateformClient.GetTargets()
		if err != nil {
			logrus.Errorf("SimpleNetworkService: err GetTargets: %v", err)
			return
		}
		for _, target := range targets {
			lbTarget, err := sns.parseLoadBalancerTarget(target)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: parseLoadBalancerTarget err: %v", err)
				continue
			}
			if len(intf.GetLocalPrefixes()) <= 0 {
				continue
			}
			contains, err := sns.prefixContainsIP(intf.GetLocalPrefixes()[0], lbTarget.GetIP())
			if !contains || err != nil {
				continue
			}
			if !sns.loadbalancer.TargetExists(lbTarget) {
				err = sns.loadbalancer.AddTarget(lbTarget)
				if err != nil {
					logrus.Errorf("SimpleNetworkService: err AddTarget (%v): %v", lbTarget, err)
				}
			}
		}
	}()
}

// InterfaceDeleted -
func (sns *SimpleNetworkService) InterfaceDeleted(intf networking.Iface) {
	for _, lbTarget := range sns.loadbalancer.GetTargets() {
		contains, err := sns.prefixContainsIP(intf.GetLocalPrefixes()[0], lbTarget.GetIP())
		if contains && err == nil {
			err := sns.loadbalancer.RemoveTarget(lbTarget)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: err RemoveTarget (%v): %v", lbTarget, err)
			}
		}
	}
}

func (sns *SimpleNetworkService) prefixContainsIP(prefix string, ip string) (bool, error) {
	prefixAddr, err := netlink.ParseAddr(prefix)
	if err != nil {
		return false, err
	}
	ipAddr, err := netlink.ParseAddr(ip)
	if err != nil {
		return false, err
	}
	return prefixAddr.Contains(ipAddr.IP), nil
}

// NewSimpleNetworkService -
func NewSimpleNetworkService(vip string, networkServicePlateformClient *nsp.NetworkServicePlateformClient, netUtils networking.Utils) *SimpleNetworkService {
	loadbalancer, err := loadbalancer.NewLoadBalancer(vip, 9973, 100, netUtils)
	if err != nil {
		logrus.Errorf("SimpleNetworkService: NewLoadBalancer err: %v", err)
	}
	err = loadbalancer.Start()
	if err != nil {
		logrus.Errorf("SimpleNetworkService: LoadBalancer start err: %v", err)
	}
	simpleNetworkService := &SimpleNetworkService{
		loadbalancer:                  loadbalancer,
		vip:                           vip,
		networkServicePlateformClient: networkServicePlateformClient,
	}
	return simpleNetworkService
}
