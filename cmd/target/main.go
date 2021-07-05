package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/nordix/meridio/pkg/health"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/target"
	"github.com/sirupsen/logrus"
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

	healthChecker, err := health.NewChecker(8000)
	if err != nil {
		logrus.Fatalf("Unable to create Health checker: %v", err)
	}
	go func() {
		err := healthChecker.Start()
		if err != nil {
			logrus.Fatalf("Unable to start Health checker: %v", err)
		}
	}()

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}

	targetConfig := target.NewConfig(config.ConfigMapName, config.NSPServiceName, config.NSPServicePort, netUtils, apiClientConfig)
	ambassador, err := target.NewAmbassador(7779, config.Namespace, targetConfig)
	if err != nil {
		logrus.Fatalf("Error creating new ambassador: %v", err)
	}
	err = ambassador.Start(ctx)
	if err != nil {
		logrus.Fatalf("Error starting ambassador: %v", err)
	}
	err = ambassador.Delete()
	if err != nil {
		logrus.Fatalf("Error deleting ambassador: %v", err)
	}

	<-ctx.Done()
}
