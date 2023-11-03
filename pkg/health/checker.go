/*
Copyright (c) 2021-2023 Nordix Foundation

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

package health

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"sync"

	"github.com/nordix/meridio/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type serviceSetElem struct{}
type serviceSet map[string]serviceSetElem

var serviceMember serviceSetElem

type HealthServerStatusModifier interface {
	SetServingStatus(service string, servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus)
	Shutdown()
}

type Checker struct {
	grpc_health_v1.HealthServer
	HealthServerStatusModifier
	ctx                context.Context
	listener           net.Listener
	server             *grpc.Server
	address            *url.URL
	probeToSubServices map[string]serviceSet // set of subservices associated with a probe service
	subServiceToProbes map[string]serviceSet // set of probe services associated with a subservice
	mu                 sync.RWMutex
}

// RegisterServices -
// Registers subservices for a probe service whose serving status will
// be deterimined by the serving status of the subservices.
// Notes:
//   - A subservice can be part of multiple probe services
//   - No check for duplicate registration (e.g. services previously
//     registered for the same probe service are not removed)
func (c *Checker) RegisterServices(probeSvc string, services ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger := log.FromContextOrGlobal(c.ctx)
	logger.V(1).Info(
		"Health server: Register subservices", "probeSvc", probeSvc,
		"services", services)

	subServices, ok := c.probeToSubServices[probeSvc]
	if !ok { // new probe service
		subServices = make(serviceSet)
		c.probeToSubServices[probeSvc] = subServices
	}
	c.HealthServerStatusModifier.SetServingStatus(probeSvc, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	for _, s := range services {
		subServices[s] = serviceMember // add subservices

		// update map linking subservice to all related probe service
		probeServices, ok := c.subServiceToProbes[s]
		if !ok { // new subservice
			probeServices = make(serviceSet)
			c.subServiceToProbes[s] = probeServices
		}
		probeServices[probeSvc] = serviceMember
		c.HealthServerStatusModifier.SetServingStatus(s, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}
}

// Check -
// Re-implements Check() function of grpc_health_v1.HealthServer interface
// in order to log if a probe service is about to fail.
func (c *Checker) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	var resp *grpc_health_v1.HealthCheckResponse
	var err error

	logDetails := func(ctx context.Context) {
		// gather all related subservice' status to provide details on the failed probe
		var svcInfo string

		c.mu.Lock()
		defer c.mu.Unlock()
		if len(c.probeToSubServices[in.Service]) > 0 {
			svcInfo += `, subservices:[`
			for s := range c.probeToSubServices[in.Service] {
				select {
				case <-ctx.Done():
					return
				default:
					svcInfo += fmt.Sprintf("%v:", s)
					status := `N/A,`
					if resp, err := c.HealthServer.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: s}); err == nil {
						status = fmt.Sprintf("%v,", resp.Status)
					}
					svcInfo += fmt.Sprintf("%v", status)
				}
			}
			svcInfo += `]`
		}
		logger := log.FromContextOrGlobal(c.ctx)
		logger.Info(
			"Probe service", "Service", in.Service,
			"Status", resp.Status, "svcInfo", svcInfo)
	}

	switch in.Service {
	case Startup:
		fallthrough
	case Readiness:
		fallthrough
	case Liveness:
		resp, err = c.HealthServer.Check(ctx, in)
		if err == nil && resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
			// gather all related subservice' status to provide details on the failed probe
			logDetails(ctx)
		}
	default:
		resp, err = c.HealthServer.Check(ctx, in)
	}

	if err != nil {
		err = fmt.Errorf("%v check failed: %w", in.Service, err)
	}
	return resp, err
}

// Re-implements SetServingStatus function in order to evaluate probe service status
// in case a related subservice is subject to the call
func (c *Checker) SetServingStatus(service string, servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch service {
	case Startup:
		fallthrough
	case Readiness:
		fallthrough
	case Liveness:
		// for probe service (Startup/Readiness/Liveness) only process serving status if no subservices
		if subServices, ok := c.probeToSubServices[service]; ok && len(subServices) > 0 {
			break
		}
		c.HealthServerStatusModifier.SetServingStatus(service, servingStatus)
	default:
		{
			c.HealthServerStatusModifier.SetServingStatus(service, servingStatus)
			// determine serving status of all related probe service if any
			if probeServices, ok := c.subServiceToProbes[service]; ok {
				for probeSvc := range probeServices {
					// belongs to a probe service (i.e. it's a subservice)
					if subServices, ok := c.probeToSubServices[probeSvc]; ok { // probe was linked to service so it must have subservices
						// aggregate serving status of subservices to get the probe status
						// (if status of input service is not SERVING no need to check subservices)
						probeStatus := grpc_health_v1.HealthCheckResponse_NOT_SERVING
						if grpc_health_v1.HealthCheckResponse_SERVING == servingStatus {
							probeStatus = grpc_health_v1.HealthCheckResponse_SERVING
							for subService := range subServices {
								// a subservice is not in SERVING status, the probe service must reflect it
								resp, err := c.HealthServer.Check(c.ctx, &grpc_health_v1.HealthCheckRequest{Service: subService})
								if err != nil || (err == nil && resp.Status != grpc_health_v1.HealthCheckResponse_SERVING) {
									probeStatus = grpc_health_v1.HealthCheckResponse_NOT_SERVING
									break
								}
							}
						}
						c.HealthServerStatusModifier.SetServingStatus(probeSvc, probeStatus)
					}
				}
			}
		}
	}
}

// Start -
// Starts gRPC health server
func (c *Checker) Start() error {

	defer func() {
		_ = c.listener.Close()
	}()
	// montior context in separate goroutine to be able to stop server
	go func() {
		<-c.ctx.Done()
		c.server.Stop()
	}()
	err := c.server.Serve(c.listener)
	if err != nil {
		return fmt.Errorf("grpc health server serve failed: %w", err)
	}
	return nil
}

// NewChecker -
// Creates gRPC health server
func NewChecker(options ...Option) (*Checker, error) {
	const unix = "unix"
	const tcp = "tcp"

	opts := &checkerOptions{
		ctx: context.Background(),
		u: func() *url.URL {
			u, _ := url.Parse(DefaultURL)
			return u
		}(),
	}
	for _, opt := range options {
		opt(opts)
	}
	// parse URL
	address := opts.u
	network, target := func(u *url.URL) (network, target string) {
		network = tcp
		target = u.Host
		if u.Scheme == unix {
			network = unix
			target = u.Path
			if target == "" {
				target = u.Opaque
			}
		}
		return network, target
	}(address)

	if network == unix {
		// remove possible lingering unix socket
		err := os.Remove(target)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to remove existing socket file: %w", err)
		}
		// create path if needed
		basePath := path.Dir(target)
		if _, err = os.Stat(basePath); os.IsNotExist(err) {
			if err = os.MkdirAll(basePath, os.ModePerm); err != nil {
				return nil, fmt.Errorf("failed to create path for %v: %w", target, err)
			}
		}
	}
	// create listener for the URL
	lis, err := net.Listen(network, target)
	if err != nil {
		return nil, fmt.Errorf("grpc health server listen failed: %w", err)
	}

	s := grpc.NewServer()
	hs := health.NewServer()

	checker := &Checker{
		HealthServer:               hs,
		HealthServerStatusModifier: hs,
		listener:                   lis,
		server:                     s,
		address:                    address,
		ctx:                        opts.ctx,
		probeToSubServices:         make(map[string]serviceSet),
		subServiceToProbes:         make(map[string]serviceSet),
	}

	grpc_health_v1.RegisterHealthServer(s, checker)

	return checker, nil
}
