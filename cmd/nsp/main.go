package main

import (
	"flag"
	"os"
	"strconv"

	"github.com/nordix/meridio/pkg/nsp"
	"github.com/sirupsen/logrus"
)

func main() {
	flag.Parse()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	port, err := strconv.Atoi(os.Getenv("NSP_PORT"))
	if err != nil || port <= 0 {
		port = 7778
	}

	nsps, _ := nsp.NewNetworkServicePlateformService(port)

	nsps.Start()
}
