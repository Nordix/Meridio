package main

import (
	"flag"
	"os"
	"strconv"

	"github.com/nordix/meridio/pkg/ipam"
	"github.com/sirupsen/logrus"
)

func main() {
	flag.Parse()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	port, err := strconv.Atoi(os.Getenv("IPAM_PORT"))
	if err != nil || port <= 0 {
		port = 7777
	}

	i, _ := ipam.NewIpamService(port)

	i.Start()
}
