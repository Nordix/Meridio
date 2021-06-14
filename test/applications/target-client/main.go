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
  request [arguments]
      Request connection to a specific network service
  close [arguments]
      Close connection to a specific network service
  show
      Show connections
`
)

func main() {
	requestCommand := flag.NewFlagSet("request", flag.ExitOnError)
	closeCommand := flag.NewFlagSet("close", flag.ExitOnError)
	showCommand := flag.NewFlagSet("show", flag.ExitOnError)

	networkServiceRequest := requestCommand.String("ns", "load-balancer", "Network Service to request connectivity")
	networkServiceClose := closeCommand.String("ns", "load-balancer", "Network Service to close connectivity")

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
	case "request":
		err := requestCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error Request Parse: %v", err.Error())
			os.Exit(1)
		}
		err = request(*networkServiceRequest)
		if err != nil {
			fmt.Printf("Error Request: %v", err.Error())
			os.Exit(1)
		}
	case "close":
		err := closeCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error Close Parse: %v", err.Error())
			os.Exit(1)
		}
		err = close(*networkServiceClose)
		if err != nil {
			fmt.Printf("Error Close: %v", err.Error())
		}
	case "show":
		err := showCommand.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("Error Close Parse: %v", err.Error())
			os.Exit(1)
		}
		err = show()
		if err != nil {
			fmt.Printf("Error Show: %v", err.Error())
		}
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

}

func request(networkService string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Request(context.Background(), &targetAPI.Connection{
		NetworkServiceName: networkService,
	})
	return err
}

func close(networkService string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	_, err = client.Close(context.Background(), &targetAPI.Connection{
		NetworkServiceName: networkService,
	})
	return err
}

func show() error {
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
