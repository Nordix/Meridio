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

	tapAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"google.golang.org/grpc"
)

const (
	usage = `usage: 
  open [arguments]
      open a stream
  close [arguments]
      Close a stream
  watch 
      Watch stream changes
`
)

func main() {
	openCommand := flag.NewFlagSet("open", flag.ExitOnError)
	closeCommand := flag.NewFlagSet("close", flag.ExitOnError)
	watchCommand := flag.NewFlagSet("watch", flag.ExitOnError)

	streamOpen := openCommand.String("s", "", "Name of the stream to open")
	networkServiceOpen := openCommand.String("c", "", "Network Service of the stream to open")
	trenchOpen := openCommand.String("t", "", "Trench of the network Service of the stream to open")

	streamClose := closeCommand.String("s", "", "Name of the stream to close")
	networkServiceClose := closeCommand.String("c", "load-balancer", "Network Service of the stream to close")
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
	streamToWatch := &nspAPI.Stream{}
	watchStreamClient, err := client.Watch(ctx, streamToWatch)
	if err != nil {
		return err
	}
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
			for _, stream := range streamResponse.GetStreamStatus() {
				fmt.Printf("%v - %v - %v - %v\n",
					stream.GetStatus(),
					stream.GetStream().GetName(),
					stream.GetStream().GetConduit().GetName(),
					stream.GetStream().GetConduit().GetTrench().GetName())
			}
		}
	}()
	<-ctx.Done()
	return nil
}

func getClient() (tapAPI.TapClient, error) {
	conn, err := grpc.Dial(os.Getenv("MERIDIO_AMBASSADOR_SOCKET"), grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		return nil, err
	}
	return tapAPI.NewTapClient(conn), nil
}
