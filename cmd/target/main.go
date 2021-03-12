package main

import (
	"context"
	"hash/fnv"
	"os"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	var config Config
	err := envconfig.Process("nsm", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	// todo
	// temporary solution
	// Wait for the proxies to be created
	time.Sleep(35 * time.Second)

	nspClient, _ := nsp.NewNetworkServicePlateformClient(config.NSPService)
	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	st := &SimpleTarget{
		networkServicePlateformClient: nspClient,
		identifier:                    identifier,
	}

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	apiClient := nsm.NewAPIClient(ctx, apiClientConfig)

	client := client.NewNetworkServiceClient(config.ProxyNetworkServiceName, apiClient)
	client.InterfaceMonitorSubscriber = st
	client.Request()

	for {
		time.Sleep(10 * time.Second)
	}
}

// Hash -
func Hash(s string, n int) int {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		logrus.Fatalf("error hashing: %+v", err)
	}
	return int(h.Sum32())%n + 1
}

// SimpleTarget -
type SimpleTarget struct {
	networkServicePlateformClient *nsp.NetworkServicePlateformClient
	identifier                    int
}

// InterfaceCreated -
func (st *SimpleTarget) InterfaceCreated(intf *networking.Interface) {
	context := make(map[string]string)
	context["identifier"] = strconv.Itoa(st.identifier)
	err := st.networkServicePlateformClient.Register(intf.LocalIPs[0].String(), context)
	if err != nil {
		logrus.Errorf("SimpleTarget: Register err: %v", err)
	}
}

// InterfaceDeleted -
func (st *SimpleTarget) InterfaceDeleted(intf *networking.Interface) {
	err := st.networkServicePlateformClient.Unregister(intf.LocalIPs[0].String())
	if err != nil {
		logrus.Errorf("SimpleTarget: Unregister err: %v", err)
	}
}
