package main

import (
	"flag"
	"os"
	"strconv"

	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/sirupsen/logrus"
)

func main() {
	flag.Parse()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	healthChecker, err := health.NewChecker(8000)
	if err != nil {
		logrus.Fatalf("Unable create Health checker: %v", err)
	}
	go healthChecker.Start()

	port, err := strconv.Atoi(os.Getenv("NSP_PORT"))
	if err != nil || port <= 0 {
		port = 7778
	}

	nsps, _ := nsp.NewNetworkServicePlateformService(port)

	nsps.Start()
}
