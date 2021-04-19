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
	"github.com/nordix/meridio/pkg/loadbalancer"
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

	nspClient, err := nsp.NewNetworkServicePlateformClient(config.NSPService)
	if err != nil {
		logrus.Errorf("NewNetworkServicePlateformClient: %v", err)
	}
	sns := NewSimpleNetworkService(vip, nspClient)

	linkMonitor, err := networking.NewLinkMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}
	interfaceMonitorEndpoint := nsm.NewInterfaceMonitorEndpoint(sns)
	linkMonitor.Subscribe(interfaceMonitorEndpoint)

	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		nsm.NewInterfaceNameEndpoint(),
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
	vip                                  *netlink.Addr
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
			logrus.Infof("SimpleNetworkService: Add Target: %v", target)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: err AddTarget (%v): %v", target, err)
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
	ip, err := netlink.ParseAddr(target.Ip)
	if err != nil {
		logrus.Errorf("SimpleNetworkService: cannot parse IP (%v): %v", target.Ip, err)
		return nil, err
	}
	return loadbalancer.NewTarget(identifier, ip), nil
}

// InterfaceCreated -
func (sns *SimpleNetworkService) InterfaceCreated(intf *networking.Interface) {
	// todo
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
			if len(intf.LocalIPs) >= 1 && !intf.LocalIPs[0].Contains(lbTarget.GetIP().IP) {
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
func (sns *SimpleNetworkService) InterfaceDeleted(intf *networking.Interface) {
	for _, lbTarget := range sns.loadbalancer.GetTargets() {
		if intf.LocalIPs[0].Contains(lbTarget.GetIP().IP) {
			err := sns.loadbalancer.RemoveTarget(lbTarget)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: err RemoveTarget (%v): %v", lbTarget, err)
			}
		}
	}
}

// NewSimpleNetworkService -
func NewSimpleNetworkService(vip *netlink.Addr, networkServicePlateformClient *nsp.NetworkServicePlateformClient) *SimpleNetworkService {
	loadbalancer, err := loadbalancer.NewLoadBalancer(vip, 9973, 100)
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
