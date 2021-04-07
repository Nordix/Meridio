package nsp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
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
	targets        sync.Map // map[string]*nspAPI.Target
	monitorStreams sync.Map // map[nspAPI.NetworkServicePlateformService_MonitorServer]bool
}

func (nsps *NetworkServicePlateformService) targetExists(target *nspAPI.Target) bool {
	_, exists := nsps.targets.Load(target.Ip)
	return exists
}

func (nsps *NetworkServicePlateformService) addTarget(target *nspAPI.Target) error {
	if nsps.targetExists(target) {
		return errors.New("Target already exists")
	}
	logrus.Infof("Add Target: %v", target)
	target.Status = nspAPI.Status_Register
	nsps.notifyMonitorStreams(target)
	nsps.targets.Store(target.Ip, target)
	return nil
}

func (nsps *NetworkServicePlateformService) removeTarget(target *nspAPI.Target) error {
	if !nsps.targetExists(target) {
		return errors.New("Target is not existing")
	}
	logrus.Infof("Remove Target: %v", target)
	target.Status = nspAPI.Status_Unregister
	nsps.notifyMonitorStreams(target)
	nsps.targets.Delete(target.Ip)
	return nil
}

func (nsps *NetworkServicePlateformService) getTargetSlice() []*nspAPI.Target {
	targets := []*nspAPI.Target{}
	nsps.targets.Range(func(key interface{}, value interface{}) bool {
		targets = append(targets, value.(*nspAPI.Target))
		return true
	})
	return targets
}

func (nsps *NetworkServicePlateformService) notifyMonitorStreams(target *nspAPI.Target) {
	nsps.monitorStreams.Range(func(key interface{}, value interface{}) bool {
		nsps.notifyMonitorStream(key.(nspAPI.NetworkServicePlateformService_MonitorServer), target)
		return true
	})
}

func (nsps *NetworkServicePlateformService) notifyMonitorStream(stream nspAPI.NetworkServicePlateformService_MonitorServer, target *nspAPI.Target) {
	if !nsps.streamAlive(stream) {
		return
	}
	err := stream.Send(target)
	if err != nil {
		nsps.monitorStreams.Store(stream, false)
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
	nsps.monitorStreams.Store(stream, true)
	nsps.targets.Range(func(key interface{}, value interface{}) bool {
		nsps.notifyMonitorStream(stream, value.(*nspAPI.Target))
		return true
	})
	for nsps.streamAlive(stream) {
		time.Sleep(1 * time.Second)
	}
	nsps.monitorStreams.Delete(stream)
	return nil
}

func (nsps *NetworkServicePlateformService) streamAlive(stream nspAPI.NetworkServicePlateformService_MonitorServer) bool {
	value, ok := nsps.monitorStreams.Load(stream)
	return ok && value.(bool)
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
		Listener: lis,
		Server:   s,
		Port:     port,
	}

	nspAPI.RegisterNetworkServicePlateformServiceServer(s, networkServicePlateformService)

	return networkServicePlateformService, nil
}
