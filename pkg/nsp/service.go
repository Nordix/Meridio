package nsp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	nspAPI "github.com/nordix/meridio/api/nsp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type NetworkServicePlateformService struct {
	Listener       net.Listener
	Server         *grpc.Server
	Port           int
	targets        map[string]*nspAPI.Target
	monitorStreams map[nspAPI.NetworkServicePlateformService_MonitorServer]bool
}

func (nsps *NetworkServicePlateformService) targetExists(target *nspAPI.Target) bool {
	_, exists := nsps.targets[target.Ip]
	return exists
}

func (nsps *NetworkServicePlateformService) addTarget(target *nspAPI.Target) error {
	if nsps.targetExists(target) {
		return errors.New("Target already exists")
	}
	target.Status = nspAPI.Status_Register
	nsps.notifyMonitorStreams(target)
	nsps.targets[target.Ip] = target
	return nil
}

func (nsps *NetworkServicePlateformService) removeTarget(target *nspAPI.Target) error {
	if nsps.targetExists(target) == false {
		return errors.New("Target is not already existing")
	}
	target.Status = nspAPI.Status_Unregister
	nsps.notifyMonitorStreams(target)
	delete(nsps.targets, target.Ip)
	return nil
}

func (nsps *NetworkServicePlateformService) getTargetSlice() []*nspAPI.Target {
	targets := []*nspAPI.Target{}
	for _, target := range nsps.targets {
		targets = append(targets, target)
	}
	return targets
}

func (nsps *NetworkServicePlateformService) notifyMonitorStreams(target *nspAPI.Target) {
	for stream := range nsps.monitorStreams {
		nsps.notifyMonitorStream(stream, target)
	}
}

func (nsps *NetworkServicePlateformService) notifyMonitorStream(stream nspAPI.NetworkServicePlateformService_MonitorServer, target *nspAPI.Target) {
	if nsps.monitorStreams[stream] == false {
		return
	}
	err := stream.Send(target)
	if err != nil {
		nsps.monitorStreams[stream] = false
	}
}

func (nsps *NetworkServicePlateformService) Register(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	err := nsps.addTarget(target)
	return &empty.Empty{}, err
}

func (nsps *NetworkServicePlateformService) Unregister(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	err := nsps.removeTarget(target)
	return &empty.Empty{}, err
}

func (nsps *NetworkServicePlateformService) Monitor(empty *empty.Empty, stream nspAPI.NetworkServicePlateformService_MonitorServer) error {
	nsps.monitorStreams[stream] = true
	for _, target := range nsps.targets {
		nsps.notifyMonitorStream(stream, target)
	}
	for nsps.monitorStreams[stream] {
		time.Sleep(1 * time.Second)
	}
	delete(nsps.monitorStreams, stream)
	return nil
}

func (nsps *NetworkServicePlateformService) GetTargets(ctx context.Context, target *empty.Empty) (*nspAPI.GetTargetsResponse, error) {
	response := &nspAPI.GetTargetsResponse{
		Targets: nsps.getTargetSlice(),
	}
	return response, nil
}

// Start -
func (nsps *NetworkServicePlateformService) Start() {
	logrus.Infof("NSP Service: Start the service (port: %v)", nsps.Port)
	if err := nsps.Server.Serve(nsps.Listener); err != nil {
		logrus.Errorf("NSP Service: failed to serve: %v", err)
	}
}

// NewNetworkServicePlateformService -
func NewNetworkServicePlateformService(port int) (*NetworkServicePlateformService, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", strconv.Itoa(port)))
	if err != nil {
		logrus.Errorf("NSP Service: failed to listen: %v", err)
		return nil, err
	}

	s := grpc.NewServer()

	networkServicePlateformService := &NetworkServicePlateformService{
		Listener:       lis,
		Server:         s,
		Port:           port,
		targets:        make(map[string]*nspAPI.Target),
		monitorStreams: make(map[nspAPI.NetworkServicePlateformService_MonitorServer]bool),
	}

	nspAPI.RegisterNetworkServicePlateformServiceServer(s, networkServicePlateformService)

	return networkServicePlateformService, nil
}
