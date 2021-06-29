package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	targetAPI "github.com/nordix/meridio/api/target"
	"google.golang.org/grpc"
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
`
)

func main() {
	connectCommand := flag.NewFlagSet("connect", flag.ExitOnError)
	disconnectCommand := flag.NewFlagSet("disconnect", flag.ExitOnError)
	requestCommand := flag.NewFlagSet("request", flag.ExitOnError)
	closeCommand := flag.NewFlagSet("close", flag.ExitOnError)

	networkServiceConnect := connectCommand.String("ns", "load-balancer", "Network Service to connect conduit")
	trenchConnect := connectCommand.String("t", "", "Trench of the network Service to connect conduit")
	trenchNamespaceConnect := connectCommand.String("tns", "", "Trench namespace of the network Service to connect conduit")

	networkServiceDisconnect := disconnectCommand.String("ns", "load-balancer", "Network Service to disconnect conduit")
	trenchDisconnect := disconnectCommand.String("t", "", "Trench of the network Service to disconnect conduit")
	trenchNamespaceDisconnect := disconnectCommand.String("tns", "", "Trench namespace of the network Service to disconnect conduit")

	networkServiceRequest := requestCommand.String("ns", "", "Network Service of the stream to request")
	trenchRequest := requestCommand.String("t", "", "Trench of the network Service of the stream to request")
	trenchNamespaceRequest := requestCommand.String("tns", "", "Trench namespace of the network Service of the stream to request")

	networkServiceClose := closeCommand.String("ns", "load-balancer", "Network Service of the stream to close")
	trenchClose := closeCommand.String("t", "", "Trench of the network Service of the stream to close")
	trenchNamespaceClose := closeCommand.String("tns", "", "Trench namespace of the network Service of the stream to close")

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
		err = connect(*networkServiceConnect, *trenchConnect, *trenchNamespaceConnect)
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
		err = disconnect(*networkServiceDisconnect, *trenchDisconnect, *trenchNamespaceDisconnect)
		if err != nil {
			fmt.Printf("Error disconnect: %v", err.Error())
		}
	case "request":
		err := requestCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error request Parse: %v", err.Error())
			os.Exit(1)
		}
		err = request(*networkServiceRequest, *trenchRequest, *trenchNamespaceRequest)
		if err != nil {
			fmt.Printf("Error request: %v", err.Error())
		}
	case "close":
		err := closeCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error close Parse: %v", err.Error())
			os.Exit(1)
		}
		err = close(*networkServiceClose, *trenchClose, *trenchNamespaceClose)
		if err != nil {
			fmt.Printf("Error close: %v", err.Error())
		}
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

}

func connect(networkService string, trench string, namespace string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Connect(context.Background(), &targetAPI.Conduit{
		NetworkServiceName: networkService,
		Trench: &targetAPI.Trench{
			Name:      trench,
			Namespace: namespace,
		},
	})
	return err
}

func disconnect(networkService string, trench string, namespace string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Disconnect(context.Background(), &targetAPI.Conduit{
		NetworkServiceName: networkService,
		Trench: &targetAPI.Trench{
			Name:      trench,
			Namespace: namespace,
		},
	})
	return err
}

func request(networkService string, trench string, namespace string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Request(context.Background(), &targetAPI.Stream{
		Conduit: &targetAPI.Conduit{
			NetworkServiceName: networkService,
			Trench: &targetAPI.Trench{
				Name:      trench,
				Namespace: namespace,
			},
		},
	})
	return err
}

func close(networkService string, trench string, namespace string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Close(context.Background(), &targetAPI.Stream{
		Conduit: &targetAPI.Conduit{
			NetworkServiceName: networkService,
			Trench: &targetAPI.Trench{
				Name:      trench,
				Namespace: namespace,
			},
		},
	})
	return err
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
