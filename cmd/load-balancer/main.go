package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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
	"github.com/nordix/meridio/pkg/configuration"
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
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	var config Config
	err := envconfig.Process("nsm", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	netUtils := &linuxKernel.KernelUtils{}

	nspClient, err := nsp.NewNetworkServicePlateformClient(config.NSPService)
	if err != nil {
		logrus.Errorf("NewNetworkServicePlateformClient: %v", err)
	}
	sns := NewSimpleNetworkService(config.VIPs, nspClient, netUtils)

	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}
	interfaceMonitorEndpoint := interfacemonitor.NewServer(interfaceMonitor, sns, netUtils)

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

	configWatcher := make(chan *configuration.Config)
	configurationWatcher := configuration.NewWatcher(config.ConfigMapName, config.Namespace, configWatcher)
	go configurationWatcher.Start()

	for {
		select {
		case config := <-configWatcher:
			sns.SetVIPs(config.VIPs)
		case <-ctx.Done():
			return
		}
	}

}

// SimpleNetworkService -
type SimpleNetworkService struct {
	loadbalancer                         *loadbalancer.LoadBalancer
	networkServicePlateformClient        *nsp.NetworkServicePlateformClient
	vips                                 []string
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
		return nil, errors.New("identifier does not exist")
	}
	identifier, err := strconv.Atoi(identifierStr)
	if err != nil {
		logrus.Errorf("SimpleNetworkService: cannot parse identifier (%v): %v", identifierStr, err)
		return nil, err
	}
	return loadbalancer.NewTarget(identifier, target.Ips), nil
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
			contains, err := sns.prefixesContainsIPs(intf.GetLocalPrefixes(), lbTarget.GetIPs())
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
		contains, err := sns.prefixesContainsIPs(intf.GetLocalPrefixes(), lbTarget.GetIPs())
		if contains && err == nil {
			err := sns.loadbalancer.RemoveTarget(lbTarget)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: err RemoveTarget (%v): %v", lbTarget, err)
			}
		}
	}
}

func (sns *SimpleNetworkService) prefixesContainsIPs(prefixes []string, ips []string) (bool, error) {
	for _, ip := range ips {
		containedInPrefixes := false
		for _, prefix := range prefixes {
			contains, err := sns.prefixContainsIP(prefix, ip)
			if err != nil {
				return false, err
			}
			if contains {
				containedInPrefixes = true
				break
			}
		}
		if !containedInPrefixes {
			return false, nil
		}
	}
	return true, nil
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

func (sns *SimpleNetworkService) SetVIPs(vips []string) {
	sns.loadbalancer.SetVIPs(vips)
}

// NewSimpleNetworkService -
func NewSimpleNetworkService(vips []string, networkServicePlateformClient *nsp.NetworkServicePlateformClient, netUtils networking.Utils) *SimpleNetworkService {
	loadbalancer, err := loadbalancer.NewLoadBalancer(vips, 9973, 100, netUtils)
	if err != nil {
		logrus.Errorf("SimpleNetworkService: NewLoadBalancer err: %v", err)
	}
	err = loadbalancer.Start()
	if err != nil {
		logrus.Errorf("SimpleNetworkService: LoadBalancer start err: %v", err)
	}
	simpleNetworkService := &SimpleNetworkService{
		loadbalancer:                  loadbalancer,
		vips:                          vips,
		networkServicePlateformClient: networkServicePlateformClient,
	}
	return simpleNetworkService
}
