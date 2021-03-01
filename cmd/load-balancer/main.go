package main

import (
	"context"
	"net"
	"os"

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
	"github.com/nordix/nvip/pkg/endpoint"
	"github.com/nordix/nvip/pkg/ipam"
	"github.com/sirupsen/logrus"
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

	_, ipnet, err := net.ParseCIDR(config.CidrPrefix)
	if err != nil {
		log.FromContext(ctx).Fatalf("error parsing cidr: %+v", err)
	}

	ipamServiceIPPort := "ipam-service:7777"

	responderEndpoint := []networkservice.NetworkServiceServer{
		// point2pointipam.NewServer(ipnet),
		ipam.NewProxyEndpoint(ipnet, ipamServiceIPPort),
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
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
