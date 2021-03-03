package main

import (
	"context"
	"os"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/edwarnicke/signalctx"
	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/nsm"
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
	// Full Mesh client
	// ********************************************************************************
	apiClient := nsm.NewAPIClient(ctx, rootConf)
	monitor := client.NewNetworkServiceClient("proxy", apiClient)
	monitor.Request()

	for {
		time.Sleep(10 * time.Second)
	}
}
