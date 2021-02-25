package main

import (
	"context"
	"net"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/edwarnicke/signalctx"
	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/nordix/nvip/pkg/client"
	"github.com/nordix/nvip/pkg/endpoint"
	"github.com/nordix/nvip/pkg/networking"
	"github.com/nordix/nvip/pkg/nsm"
	"github.com/nordix/nvip/pkg/proxy"
	"github.com/sirupsen/logrus"
)

func main() {
	// ********************************************************************************
	// Configure signal handling context
	// ********************************************************************************
	ctx := signalctx.WithSignals(context.Background())
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	// ********************************************************************************
	// Setup logger
	// ********************************************************************************
	logrus.Info("Starting NetworkServiceMesh Client ...")
	logrus.SetFormatter(&nested.Formatter{})
	ctx = log.WithFields(ctx, map[string]interface{}{"cmd": os.Args[:1]})
	ctx = log.WithLog(ctx, logruslogger.New(ctx))

	// ********************************************************************************
	// Configure open tracing
	// ********************************************************************************
	// Enable Jaeger
	log.EnableTracing(true)
	jaegerCloser := jaeger.InitJaeger(ctx, "proxy")
	defer func() { _ = jaegerCloser.Close() }()

	// ********************************************************************************
	// Start the Proxy (NSE + NSC)
	// ********************************************************************************

	vip, _ := netlink.ParseAddr("20.0.0.1/32")

	linkMonitor, err := networking.NewLinkMonitor()
	if err != nil {
		log.FromContext(ctx).Fatalf("Error creating link monitor: %+v", err)
	}
	p := proxy.NewProxy(vip)
	proxyEndpoint := proxy.NewProxyEndpoint(p)
	linkMonitor.Subscribe(proxyEndpoint)
	go StartNSC(ctx)
	StartNSE(ctx, proxyEndpoint)
}

func StartNSC(ctx context.Context) {
	rootConf := &client.Config{}
	if err := envconfig.Usage("nsm", rootConf); err != nil {
		log.FromContext(ctx).Fatal(err)
	}
	if err := envconfig.Process("nsm", rootConf); err != nil {
		log.FromContext(ctx).Fatalf("error processing rootConf from env: %+v", err)
	}
	log.FromContext(ctx).Infof("rootConf: %+v", rootConf)

	apiClient := nsm.NewAPIClient(ctx, rootConf)
	monitor := client.NewMonitor("load-balancer", apiClient, apiClient)
	monitor.Start()
}

func StartNSE(ctx context.Context, proxyEndpoint *proxy.ProxyEndpoint) {
	// get config from environment
	config := new(endpoint.Config)
	if err := config.Process(); err != nil {
		logrus.Fatal(err.Error())
	}

	log.FromContext(ctx).Infof("Config: %#v", config)

	_, ipnet, err := net.ParseCIDR(config.CidrPrefix)
	if err != nil {
		log.FromContext(ctx).Fatalf("error parsing cidr: %+v", err)
	}

	responderEndpoint := []networkservice.NetworkServiceServer{
		point2pointipam.NewServer(ipnet),
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		proxyEndpoint,
		sendfd.NewServer(),
	}

	ep := endpoint.NewEndpoint(ctx, config)

	err = ep.Start(responderEndpoint...)
	if err != nil {
		log.FromContext(ctx).Fatalf("unable to start nse %+v", err)
	}

	defer ep.Delete()

	<-ctx.Done()
}
