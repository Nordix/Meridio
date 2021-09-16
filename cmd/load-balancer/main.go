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
	"errors"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	nspAPI "github.com/nordix/meridio/api/nsp"
	"github.com/nordix/meridio/pkg/configuration"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/health"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/loadbalancer"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	var config Config
	err := envconfig.Process("nsm", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	netUtils := &linuxKernel.KernelUtils{}

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

	nspClient, err := nsp.NewNetworkServicePlateformClient(config.NSPService)
	if err != nil {
		logrus.Errorf("NewNetworkServicePlateformClient: %v", err)
	}
	sns := NewSimpleNetworkService(ctx, nspClient, netUtils)

	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}
	interfaceMonitorEndpoint := interfacemonitor.NewServer(interfaceMonitor, sns, netUtils)

	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		interfacename.NewServer("nse", &interfacename.RandomGenerator{}),
		interfaceMonitorEndpoint,
		sendfd.NewServer(),
	}

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	nsmAPIClient := nsm.NewAPIClient(ctx, apiClientConfig)

	endpointConfig := &endpoint.Config{
		Name:             config.Name,
		ServiceName:      config.ServiceName,
		Labels:           config.Labels,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	ep, err := endpoint.NewEndpoint(ctx, endpointConfig, nsmAPIClient.NetworkServiceRegistryClient, nsmAPIClient.NetworkServiceEndpointRegistryClient)
	if err != nil {
		logrus.Fatalf("unable to create a new nse %+v", err)
	}

	err = ep.StartWithoutRegister(responderEndpoint...)
	if err != nil {
		logrus.Fatalf("unable to start nse %+v", err)
	}

	defer ep.Delete()

	sns.Start()
	// monitor availibilty of frontends; if no feasible FE don't advertise NSE to proxies
	fns := NewFrontendNetworkService(nspClient, ep, NewServiceControlDispatcher(sns))
	fns.Start()

	configWatcher := make(chan *configuration.OperatorConfig)
	configurationWatcher := configuration.NewOperatorWatcher(config.ConfigMapName, config.Namespace, configWatcher)
	go configurationWatcher.Start()

	for {
		select {
		case config := <-configWatcher:
			sns.SetVIPs(configuration.AddrListFromVipConfig(config.VIPs))
		case <-ctx.Done():
			return
		}
	}

}

// SimpleNetworkService -
type SimpleNetworkService struct {
	loadbalancer                         *loadbalancer.LoadBalancer
	networkServicePlateformClient        *nsp.NetworkServicePlateformClient
	networkServicePlateformServiceStream nspAPI.NetworkServicePlateformService_MonitorClient
	interfaces                           sync.Map
	ctx                                  context.Context
	serviceCtrCh                         chan bool
	simpleNetworkServiceBlocked          bool
	mu                                   sync.Mutex
}

// Start -
func (sns *SimpleNetworkService) Start() {
	var err error
	sns.networkServicePlateformServiceStream, err = sns.networkServicePlateformClient.Monitor()
	if err != nil {
		logrus.Errorf("SimpleNetworkService: err Monitor: %v", err)
	}
	go sns.recv()

	go func() {
		for {
			select {
			case allowService, ok := <-sns.serviceCtrCh:
				if ok {
					sns.mu.Lock()
					pfx := ""
					if allowService {
						pfx = "un"
					}
					logrus.Infof("simpleNetworkService: %vblock service (allowService=%v)", pfx, allowService)

					sns.simpleNetworkServiceBlocked = !allowService
					// When service is blocked it implies that the southbound NSE gets also removed.
					// Removal of the NSE from registry prompts the NSC side to close the related NSM
					// connections making the associated interfaces unusable. However unfortunately
					// NSM is not able to properly close a connection associated with a "disappeared" NSE
					// (so NSM interfaces remain as well).
					//
					// Thus in SimpleNetworkService we must prohibit processing of new Targets and
					// creation of new southbound NSE interfaces while NSE removal takes effect on
					// NSC side.
					// Moreover the known Targets and thus the associated routing must be force removed.
					// That's because once the "block" is lifted, the southbound NSE should be advertised
					// again, resulting in new NSM Service Requests and thus interfaces for which the Target
					// routes must be readjusted.
					// Interference of old NSM interfaces must be avoided, thus their link state is changed
					// to down. (Hopefully once NSM finally decides to remove an old interface (e.g. due
					// to some timeout or whatever) this state change won't screw up things...)
					//
					// Note: Currently SimpleNetworkServiceClient/FullMeshNetworkServiceClient on the proxy side
					// will keep trying to establish an NSM connection forever, while also blocking NSE event
					// processing. So if the NSE disappeared in the meantime, it will go unnoticed by the proxy.
					if sns.simpleNetworkServiceBlocked {
						sns.evictLoadBalancerTargets()
						sns.disableInterfaces()
					}
					sns.mu.Unlock()
				}
			case <-sns.ctx.Done():
				return
			}
		}
	}()
}

func (sns *SimpleNetworkService) recv() {
	for {
		targetEvent, err := sns.networkServicePlateformServiceStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Errorf("SimpleNetworkService: event err: %v", err)
			break
		}

		target := targetEvent.Target
		lbTarget, err := sns.parseLoadBalancerTarget(targetEvent.Target)
		if err != nil {
			logrus.Errorf("SimpleNetworkService: parseLoadBalancerTarget err: %v", err)
			continue
		}

		if (targetEvent.Status == nspAPI.TargetEvent_Register || targetEvent.Status == nspAPI.TargetEvent_Updated) && target.Status == nspAPI.Target_Enabled {
			// if service is blocked, do not process new Target
			if sns.serviceBlocked() {
				continue
			}
			err = sns.loadbalancer.AddTarget(lbTarget)
			logrus.Infof("SimpleNetworkService: Add Target: %v", target)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: err AddTarget (%v): %v", target, err)
				continue
			}
		} else if targetEvent.Status == nspAPI.TargetEvent_Unregister || (targetEvent.Status == nspAPI.TargetEvent_Updated && target.Status == nspAPI.Target_Disabled) {
			err = sns.loadbalancer.RemoveTarget(lbTarget)
			logrus.Infof("SimpleNetworkService: Remove Target: %v", target)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: err RemoveTarget (%v): %v", target, err)
				continue
			}
		}
	}
}

func (sns *SimpleNetworkService) parseLoadBalancerTarget(target *nspAPI.Target) (*loadbalancer.Target, error) {
	identifierStr, exists := target.Context[nsp.Identifier.String()]
	if !exists {
		logrus.Errorf("SimpleNetworkService: identifier does not exist: %v", target.Context)
		return nil, errors.New("identifier does not exist")
	}
	identifier, err := strconv.Atoi(identifierStr)
	if err != nil {
		logrus.Errorf("SimpleNetworkService: cannot parse identifier (%v): %v", identifierStr, err)
		return nil, err
	}
	return loadbalancer.NewTarget(identifier, target.Ips), nil
}

// InterfaceCreated -
func (sns *SimpleNetworkService) InterfaceCreated(intf networking.Iface) {
	logrus.Infof("SimpleNetworkService: InterfaceCreated: %v", intf)
	go func() {
		if sns.serviceBlocked() {
			// if service blocked, do not process new interface events (which
			// might appear until the block takes effect on NSC side)
			// instead disable them not to interfere after the block is lifted
			sns.disableInterface(intf)
			return
		}
		sns.interfaces.Store(intf.GetIndex(), intf)

		select {
		case <-sns.ctx.Done():
			return
		case <-time.After(2 * time.Second): // 2 sec passed
		}

		targets, err := sns.networkServicePlateformClient.GetTargets()
		if err != nil {
			logrus.Errorf("SimpleNetworkService: err GetTargets: %v", err)
			return
		}
		for _, target := range targets {
			lbTarget, err := sns.parseLoadBalancerTarget(target)
			if err != nil {
				logrus.Errorf("SimpleNetworkService: parseLoadBalancerTarget err: %v", err)
				continue
			}
			if len(intf.GetLocalPrefixes()) <= 0 {
				continue
			}
			if target.Status == nspAPI.Target_Disabled {
				continue
			}
			contains, err := sns.prefixesContainsIPs(intf.GetLocalPrefixes(), lbTarget.GetIPs())
			if !contains || err != nil {
				continue
			}
			if !sns.loadbalancer.TargetExists(lbTarget) {
				logrus.Infof("SimpleNetworkService: Add Target: %v", target)
				err = sns.loadbalancer.AddTarget(lbTarget)
				if err != nil {
					logrus.Errorf("SimpleNetworkService: err AddTarget (%v): %v", lbTarget, err)
				}
			} else {
				logrus.Debugf("SimpleNetworkService: InterfaceCreated: TargetExists: %v", lbTarget)
			}
		}
	}()
}

// InterfaceDeleted -
func (sns *SimpleNetworkService) InterfaceDeleted(intf networking.Iface) {
	logrus.Infof("SimpleNetworkService: InterfaceDeleted: Intf %v", intf)
	if _, ok := sns.interfaces.LoadAndDelete(intf.GetIndex()); ok {
		for _, lbTarget := range sns.loadbalancer.GetTargets() {
			contains, err := sns.prefixesContainsIPs(intf.GetLocalPrefixes(), lbTarget.GetIPs())
			if contains && err == nil {
				logrus.Infof("SimpleNetworkService: Remove Target: %v", lbTarget)
				err := sns.loadbalancer.RemoveTarget(lbTarget)
				if err != nil {
					logrus.Errorf("SimpleNetworkService: err RemoveTarget (%v): %v", lbTarget, err)
				}
			}
		}
	}
}

func (sns *SimpleNetworkService) prefixesContainsIPs(prefixes []string, ips []string) (bool, error) {
	for _, ip := range ips {
		containedInPrefixes := false
		for _, prefix := range prefixes {
			contains, err := sns.prefixContainsIP(prefix, ip)
			if err != nil {
				return false, err
			}
			if contains {
				containedInPrefixes = true
				break
			}
		}
		if !containedInPrefixes {
			return false, nil
		}
	}
	return true, nil
}

func (sns *SimpleNetworkService) prefixContainsIP(prefix string, ip string) (bool, error) {
	prefixAddr, err := netlink.ParseAddr(prefix)
	if err != nil {
		return false, err
	}
	ipAddr, err := netlink.ParseAddr(ip)
	if err != nil {
		return false, err
	}
	return prefixAddr.Contains(ipAddr.IP), nil
}

func (sns *SimpleNetworkService) SetVIPs(vips []string) {
	sns.loadbalancer.SetVIPs(vips)
}

func (sns *SimpleNetworkService) serviceBlocked() bool {
	sns.mu.Lock()
	defer sns.mu.Unlock()
	return sns.simpleNetworkServiceBlocked
}

func (sns *SimpleNetworkService) GetServiceControlChannel() interface{} {
	return (chan<- bool)(sns.serviceCtrCh)
}

func (sns *SimpleNetworkService) evictLoadBalancerTargets() {
	logrus.Infof("SimpleNetworkService: Evict Targets")
	for _, lbTarget := range sns.loadbalancer.GetTargets() {
		logrus.Debugf("SimpleNetworkService: Evict Target %v", lbTarget)
		err := sns.loadbalancer.RemoveTarget(lbTarget)
		if err != nil {
			logrus.Warnf("SimpleNetworkService: err EvictTarget (%v): %v", lbTarget, err)
		}
	}
}

// disableInterfaces -
// Set interfaces down, so that they won't interface with future "Add Target"
// operation. Meaning old interfaces not yet removed by NSM must not get associated
// with routes inserted for Targets after the block is lifted.
func (sns *SimpleNetworkService) disableInterfaces() {
	logrus.Infof("SimpleNetworkService: Disable Interfaces")
	sns.interfaces.Range(func(key interface{}, value interface{}) bool {
		sns.disableInterface(value.(networking.Iface))
		sns.interfaces.Delete(key)
		return true
	})
}

// disableInterface -
// Set interface state down
func (sns *SimpleNetworkService) disableInterface(intf networking.Iface) {
	logrus.Debugf("SimpleNetworkService: Disable Intf %v", intf)
	la := netlink.NewLinkAttrs()
	la.Index = intf.GetIndex()
	err := netlink.LinkSetDown(&netlink.Dummy{LinkAttrs: la})
	if err != nil {
		logrus.Warnf("SimpleNetworkService: err Disable Intf (%v): %v", la.Index, err)
	}
}

/* // Request checks if allowed to serve the request
// A non-nil error is returned if serving the request was rejected, or if a next element in the chain returns a non-nil error
// It implements NetworkServiceServer for SimpleNetworkService
//
// TODO: Is this feature even needed? Currently, SimpleNetworkServiceClient will keep trying to establish an NSM connection
// forever, during which it also blocks NSE event processing. So it won't notice if the NSE has disappeared in the meantime.
// Although this is a valid problem, irrespective of the fact whether SimpleNetworkService blocks Requests or not...
// Moreover generally NSM is really pushing to establish a connection on Requests, thus letting the Request through, could lead
// to a better outcome...
func (sns *SimpleNetworkService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if sns.serviceBlocked() {
		return nil, errors.New("SimpleNetworkService blocked")
	}

	logrus.Infof("SimpleNetworkService: Request")
	return next.Server(ctx).Request(ctx, request)
}

// Close it does nothing except calling the next Close in the chain
// A non-nil error if a next element in the chain returns a non-nil error
// It implements NetworkServiceServer for SimpleNetworkService
func (sns *SimpleNetworkService) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logrus.Infof("SimpleNetworkService: Close")
	return next.Server(ctx).Close(ctx, conn)
} */

// NewSimpleNetworkService -
func NewSimpleNetworkService(ctx context.Context, networkServicePlateformClient *nsp.NetworkServicePlateformClient, netUtils networking.Utils) *SimpleNetworkService {
	loadbalancer, err := loadbalancer.NewLoadBalancer([]string{}, 9973, 100, netUtils)
	if err != nil {
		logrus.Errorf("SimpleNetworkService: NewLoadBalancer err: %v", err)
	}
	err = loadbalancer.Start()
	if err != nil {
		logrus.Errorf("SimpleNetworkService: LoadBalancer start err: %v", err)
	}
	simpleNetworkService := &SimpleNetworkService{
		loadbalancer:                  loadbalancer,
		networkServicePlateformClient: networkServicePlateformClient,
		serviceCtrCh:                  make(chan bool),
		simpleNetworkServiceBlocked:   true,
		ctx:                           ctx,
	}
	return simpleNetworkService
}
