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
	"io"
	"os"
	"os/signal"
	"syscall"

	targetAPI "github.com/nordix/meridio/api/target"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	usage = `usage: 
  connect [arguments]
      Connect to a specific network service (conduit)
  disconnect [arguments]
      Disconnect from a specific network service (conduit)
  request [arguments]
      Request a stream
  close [arguments]
      Close a stream
  watch
      Watch conduit connection/disconnect events and stream request/close events
`
)

func main() {
	connectCommand := flag.NewFlagSet("connect", flag.ExitOnError)
	disconnectCommand := flag.NewFlagSet("disconnect", flag.ExitOnError)
	requestCommand := flag.NewFlagSet("request", flag.ExitOnError)
	closeCommand := flag.NewFlagSet("close", flag.ExitOnError)
	watchCommand := flag.NewFlagSet("watch", flag.ExitOnError)

	networkServiceConnect := connectCommand.String("ns", "load-balancer", "Network Service to connect conduit")
	trenchConnect := connectCommand.String("t", "", "Trench of the network Service to connect conduit")

	networkServiceDisconnect := disconnectCommand.String("ns", "load-balancer", "Network Service to disconnect conduit")
	trenchDisconnect := disconnectCommand.String("t", "", "Trench of the network Service to disconnect conduit")

	streamRequest := requestCommand.String("s", "", "Name of the stream to request")
	networkServiceRequest := requestCommand.String("ns", "", "Network Service of the stream to request")
	trenchRequest := requestCommand.String("t", "", "Trench of the network Service of the stream to request")

	streamClose := closeCommand.String("s", "", "Name of the stream to close")
	networkServiceClose := closeCommand.String("ns", "load-balancer", "Network Service of the stream to close")
	trenchClose := closeCommand.String("t", "", "Trench of the network Service of the stream to close")

	flag.Usage = func() {
		fmt.Printf("%s", usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(os.Args) < 2 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "connect":
		err := connectCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error connect Parse: %v", err.Error())
			os.Exit(1)
		}
		err = connect(*networkServiceConnect, *trenchConnect)
		if err != nil {
			fmt.Printf("Error connect: %v", err.Error())
			os.Exit(1)
		}
	case "disconnect":
		err := disconnectCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error disconnect Parse: %v", err.Error())
			os.Exit(1)
		}
		err = disconnect(*networkServiceDisconnect, *trenchDisconnect)
		if err != nil {
			fmt.Printf("Error disconnect: %v", err.Error())
		}
	case "request":
		err := requestCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error request Parse: %v", err.Error())
			os.Exit(1)
		}
		err = request(*streamRequest, *networkServiceRequest, *trenchRequest)
		if err != nil {
			fmt.Printf("Error request: %v", err.Error())
		}
	case "close":
		err := closeCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error close Parse: %v", err.Error())
			os.Exit(1)
		}
		err = close(*streamClose, *networkServiceClose, *trenchClose)
		if err != nil {
			fmt.Printf("Error close: %v", err.Error())
		}
	case "watch":
		err := watchCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error watch Parse: %v", err.Error())
			os.Exit(1)
		}
		err = watch()
		if err != nil {
			fmt.Printf("Error to watch: %v", err.Error())
		}
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

}

func connect(networkService string, trench string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Connect(context.Background(), &targetAPI.Conduit{
		NetworkServiceName: networkService,
		Trench: &targetAPI.Trench{
			Name: trench,
		},
	})
	return err
}

func disconnect(networkService string, trench string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Disconnect(context.Background(), &targetAPI.Conduit{
		NetworkServiceName: networkService,
		Trench: &targetAPI.Trench{
			Name: trench,
		},
	})
	return err
}

func request(stream string, networkService string, trench string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Request(context.Background(), &targetAPI.Stream{
		Name: stream,
		Conduit: &targetAPI.Conduit{
			NetworkServiceName: networkService,
			Trench: &targetAPI.Trench{
				Name: trench,
			},
		},
	})
	return err
}

func close(stream string, networkService string, trench string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Close(context.Background(), &targetAPI.Stream{
		Name: stream,
		Conduit: &targetAPI.Conduit{
			NetworkServiceName: networkService,
			Trench: &targetAPI.Trench{
				Name: trench,
			},
		},
	})
	return err
}

func watch() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()
	client, err := getClient()
	if err != nil {
		return err
	}
	watchConduitClient, err := client.WatchConduits(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	watchStreamClient, err := client.WatchStreams(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	go func() {
		for {
			conduitEvent, err := watchConduitClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			fmt.Printf("conduit event: %v - %v - %v\n", conduitEvent.Conduit.GetTrench().Name, conduitEvent.Conduit.NetworkServiceName, conduitEvent.ConduitEventStatus)
		}
	}()
	go func() {
		for {
			streamEvent, err := watchStreamClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			fmt.Printf("stream event: %v - %v - %v\n", streamEvent.Stream.Conduit.GetTrench().Name, streamEvent.Stream.Conduit.NetworkServiceName, streamEvent.StreamEventStatus)
		}
	}()
	<-ctx.Done()
	return nil
}

func getClient() (targetAPI.AmbassadorClient, error) {
	conn, err := grpc.Dial("localhost:7779", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		return nil, err
	}
	return targetAPI.NewAmbassadorClient(conn), nil
}
