/*
Copyright (c) 2021 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	ipamAPI "github.com/nordix/meridio/api/ipam"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/ipam"
	"github.com/nordix/meridio/pkg/security/credentials"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	flag.Parse()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

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

	port, err := strconv.Atoi(os.Getenv("IPAM_PORT"))
	if err != nil || port <= 0 {
		port = 7777
	}

	datastore := os.Getenv("IPAM_DATASOURCE")
	if datastore == "" {
		datastore = "/run/ipam/data/registry.db"
	}

	ipamServer, err := ipam.NewServer(datastore)
	if err != nil {
		logrus.Fatalf("Unable to create ipam server: %v", err)
	}

	server := grpc.NewServer(grpc.Creds(
		credentials.GetServer(context.Background()),
	))
	ipamAPI.RegisterIpamServiceServer(server, ipamServer)

	logrus.Infof("IPAM Service: Start the service (port: %v)", port)
	listener, err := net.Listen("tcp", fmt.Sprintf("[::]:%d", port))
	if err != nil {
		logrus.Fatalf("NSP Service: failed to listen: %v", err)
	}

	if err := server.Serve(listener); err != nil {
		logrus.Errorf("NSP Service: failed to serve: %v", err)
	}
}
