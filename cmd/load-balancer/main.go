package main

import (
	"context"
	"io"
	"os"
	"strconv"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/edwarnicke/debug"
	"github.com/edwarnicke/signalctx"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	nspAPI "github.com/nordix/meridio/api/nsp"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/loadbalancer"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func main() {

	// ********************************************************************************
	// setup context to catch signals
	// ********************************************************************************
	ctx := signalctx.WithSignals(context.Background())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// ********************************************************************************
	// setup logging
	// ********************************************************************************
	logrus.SetFormatter(&nested.Formatter{})
	ctx = log.WithFields(ctx, map[string]interface{}{"cmd": os.Args[0]})
	ctx = log.WithLog(ctx, logruslogger.New(ctx))

	if err := debug.Self(); err != nil {
		log.FromContext(ctx).Infof("%s", err)
	}

	// ********************************************************************************
	// Configure open tracing
	// ********************************************************************************
	log.EnableTracing(true)
	jaegerCloser := jaeger.InitJaeger(ctx, "load-balancer")
	defer func() { _ = jaegerCloser.Close() }()

	// get config from environment
	config := new(endpoint.Config)
	if err := config.Process(); err != nil {
		logrus.Fatal(err.Error())
	}

	log.FromContext(ctx).Infof("Config: %#v", config)

	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		sendfd.NewServer(),
	}

	vip, _ := netlink.ParseAddr("20.0.0.1/32")
	nspServiceIPPort := "nsp-service:7778"
	nspClient, err := nsp.NewNetworkServicePlateformClient(nspServiceIPPort)
	if err != nil {
		logrus.Errorf("NewNetworkServicePlateformClient: %v", err)
	}
	sns := NewSimpleNetworkService(vip, nspClient)

	ep := endpoint.NewEndpoint(ctx, config)

	err = ep.Start(responderEndpoint...)
	if err != nil {
		log.FromContext(ctx).Fatalf("unable to start nse %+v", err)
	}

	defer ep.Delete()

	sns.Start()

	<-ctx.Done()
}

type SimpleNetworkService struct {
	loadbalancer                         *loadbalancer.LoadBalancer
	networkServicePlateformClient        *nsp.NetworkServicePlateformClient
	vip                                  *netlink.Addr
	networkServicePlateformServiceStream nspAPI.NetworkServicePlateformService_MonitorClient
}

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
		if exists == false {
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
			sns.loadbalancer.RemoveTarget(lbTarget)
			logrus.Infof("SimpleNetworkService: Remove Target: %v", target)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: err RemoveTarget (%v): %v", target, err)
				continue
			}
		}
	}
}

func NewSimpleNetworkService(vip *netlink.Addr, networkServicePlateformClient *nsp.NetworkServicePlateformClient) *SimpleNetworkService {
	loadbalancer := loadbalancer.NewLoadBalancer(vip, 9973, 100)
	err := loadbalancer.Start()
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
