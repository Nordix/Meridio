package main

import (
	"context"
	"hash/fnv"
	"os"
	"strconv"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/edwarnicke/signalctx"
	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsp"
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
	jaegerCloser := jaeger.InitJaeger(ctx, "nsc")
	defer func() { _ = jaegerCloser.Close() }()

	// ********************************************************************************
	// Get config from environment
	// ********************************************************************************
	rootConf := &client.Config{}
	if err := envconfig.Usage("nsm", rootConf); err != nil {
		log.FromContext(ctx).Fatal(err)
	}
	if err := envconfig.Process("nsm", rootConf); err != nil {
		log.FromContext(ctx).Fatalf("error processing rootConf from env: %+v", err)
	}
	log.FromContext(ctx).Infof("rootConf: %+v", rootConf)

	// ********************************************************************************
	// Simple Target
	// ********************************************************************************

	nspServiceIPPort := "nsp-service:7778"
	nspClient, _ := nsp.NewNetworkServicePlateformClient(nspServiceIPPort)
	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	st := &SimpleTarget{
		networkServicePlateformClient: nspClient,
		identifier:                    identifier,
	}

	apiClient := nsm.NewAPIClient(ctx, rootConf)
	client := client.NewNetworkServiceClient("proxy", apiClient)
	client.InterfaceMonitorSubscriber = st
	client.Request()

	for {
		time.Sleep(10 * time.Second)
	}
}

func Hash(s string, n int) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())%n + 1
}

type SimpleTarget struct {
	networkServicePlateformClient *nsp.NetworkServicePlateformClient
	identifier                    int
}

func (st *SimpleTarget) InterfaceCreated(intf *networking.Interface) {
	context := make(map[string]string)
	context["identifier"] = strconv.Itoa(st.identifier)
	st.networkServicePlateformClient.Register(intf.LocalIPs[0].String(), context)
}

func (st *SimpleTarget) InterfaceDeleted(intf *networking.Interface) {
	st.networkServicePlateformClient.Unregister(intf.LocalIPs[0].String())
}
