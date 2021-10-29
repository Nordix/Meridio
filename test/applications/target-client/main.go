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

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	targetAPI "github.com/nordix/meridio/api/target/v1"
	"google.golang.org/grpc"
)

const (
	usage = `usage: 
  connect [arguments]
      Connect to a specific network service (conduit)
  disconnect [arguments]
      Disconnect from a specific network service (conduit)
  open [arguments]
      open a stream
  close [arguments]
      Close a stream
  watch 
      Watch conduit connection/disconnect events and stream open/close events
`
)

func main() {
	connectCommand := flag.NewFlagSet("connect", flag.ExitOnError)
	disconnectCommand := flag.NewFlagSet("disconnect", flag.ExitOnError)
	openCommand := flag.NewFlagSet("oper", flag.ExitOnError)
	closeCommand := flag.NewFlagSet("close", flag.ExitOnError)
	watchCommand := flag.NewFlagSet("watch", flag.ExitOnError)

	networkServiceConnect := connectCommand.String("ns", "load-balancer", "Network Service to connect conduit")
	trenchConnect := connectCommand.String("t", "", "Trench of the network Service to connect conduit")

	networkServiceDisconnect := disconnectCommand.String("ns", "load-balancer", "Network Service to disconnect conduit")
	trenchDisconnect := disconnectCommand.String("t", "", "Trench of the network Service to disconnect conduit")

	streamOpen := openCommand.String("s", "", "Name of the stream to open")
	networkServiceOpen := openCommand.String("ns", "", "Network Service of the stream to open")
	trenchOpen := openCommand.String("t", "", "Trench of the network Service of the stream to open")

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
			fmt.Printf("Error connect Parse: %v\n", err.Error())
			os.Exit(1)
		}
		err = connect(*networkServiceConnect, *trenchConnect)
		if err != nil {
			fmt.Printf("Error connect: %v\n", err.Error())
			os.Exit(1)
		}
	case "disconnect":
		err := disconnectCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error disconnect Parse: %v\n", err.Error())
			os.Exit(1)
		}
		err = disconnect(*networkServiceDisconnect, *trenchDisconnect)
		if err != nil {
			fmt.Printf("Error disconnect: %v\n", err.Error())
		}
	case "open":
		err := openCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error open Parse: %v\n", err.Error())
			os.Exit(1)
		}
		err = open(*streamOpen, *networkServiceOpen, *trenchOpen)
		if err != nil {
			fmt.Printf("Error open: %v\n", err.Error())
		}
	case "close":
		err := closeCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error close Parse: %v\n", err.Error())
			os.Exit(1)
		}
		err = close(*streamClose, *networkServiceClose, *trenchClose)
		if err != nil {
			fmt.Printf("Error close: %v\n", err.Error())
		}
	case "watch":
		err := watchCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error watch Parse: %v\n", err.Error())
			os.Exit(1)
		}
		err = watch()
		if err != nil {
			fmt.Printf("Error to watch: %v\n", err.Error())
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
	_, err = client.Connect(context.Background(), &nspAPI.Conduit{
		Name: networkService,
		Trench: &nspAPI.Trench{
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
	_, err = client.Disconnect(context.Background(), &nspAPI.Conduit{
		Name: networkService,
		Trench: &nspAPI.Trench{
			Name: trench,
		},
	})
	return err
}

func open(stream string, networkService string, trench string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Open(context.Background(), &nspAPI.Stream{
		Name: stream,
		Conduit: &nspAPI.Conduit{
			Name: networkService,
			Trench: &nspAPI.Trench{
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
	_, err = client.Close(context.Background(), &nspAPI.Stream{
		Name: stream,
		Conduit: &nspAPI.Conduit{
			Name: networkService,
			Trench: &nspAPI.Trench{
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
	conduitToWatch := &nspAPI.Conduit{}
	watchConduitClient, err := client.WatchConduit(ctx, conduitToWatch)
	if err != nil {
		return err
	}
	streamToWatch := &nspAPI.Stream{}
	watchStreamClient, err := client.WatchStream(ctx, streamToWatch)
	if err != nil {
		return err
	}
	go func() {
		for {
			conduitResponse, err := watchConduitClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			fmt.Printf("New conduit list:\n")
			for _, conduit := range conduitResponse.GetConduits() {
				fmt.Printf("%v - %v\n", conduit.GetName(), conduit.GetTrench().GetName())
			}
		}
	}()
	go func() {
		for {
			streamResponse, err := watchStreamClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			fmt.Printf("New stream list:\n")
			for _, stream := range streamResponse.GetStreams() {
				fmt.Printf("%v - %v - %v\n", stream.GetName(), stream.GetConduit().GetName(), stream.GetConduit().GetTrench().GetName())
			}
		}
	}()
	<-ctx.Done()
	return nil
}

func getClient() (targetAPI.AmbassadorClient, error) {
	conn, err := grpc.Dial(os.Getenv("MERIDIO_AMBASSADOR_SOCKET"), grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		return nil, err
	}
	return targetAPI.NewAmbassadorClient(conn), nil
}
