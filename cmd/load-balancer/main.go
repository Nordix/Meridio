package main

import (
	"context"
	"io"
	"os"
	"strconv"

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

	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		sendfd.NewServer(),
	}

	nspClient, err := nsp.NewNetworkServicePlateformClient(config.NSPService)
	if err != nil {
		logrus.Errorf("NewNetworkServicePlateformClient: %v", err)
	}
	sns := NewSimpleNetworkService(vip, nspClient)

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
	ep := endpoint.NewEndpoint(ctx, endpointConfig, nsmAPIClient.NetworkServiceRegistryClient, nsmAPIClient.NetworkServiceEndpointRegistryClient)

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

		identifierStr, exists := target.Context["identifier"]
		if !exists {
			logrus.Errorf("SimpleNetworkService: identifier does not exist: %v", target.Context)
			continue
		}
		identifier, err := strconv.Atoi(identifierStr)
		if err != nil {
			logrus.Errorf("SimpleNetworkService: cannot parse identifier (%v): %v", identifierStr, err)
			continue
		}
		ip, err := netlink.ParseAddr(target.Ip)
		if err != nil {
			logrus.Errorf("SimpleNetworkService: cannot parse IP (%v): %v", target.Ip, err)
			continue
		}

		lbTarget := loadbalancer.NewTarget(identifier, ip)

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
