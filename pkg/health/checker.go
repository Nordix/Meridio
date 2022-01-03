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

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type HealthServerStatusModifier interface {
	SetServingStatus(service string, servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus)
	Shutdown()
}

type Checker struct {
	grpc_health_v1.HealthServer
	HealthServerStatusModifier
	ctx               context.Context
	listener          net.Listener
	server            *grpc.Server
	address           *url.URL
	subServices       map[string][]string // subservices associated with a probe service
	subServiceToProbe map[string]string   // map subservice to probe service
	mu                sync.RWMutex
}

// RegisterServices -
// Registers subservices for a probe service whose serving status will
// be deterimined by the serving status of the subservices.
// Note: currently a subservice can only belong to one probe service
func (c *Checker) RegisterServices(probeSvc string, services ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logrus.Debugf("Health server: Register subservices for probeSvc=%v, services=%v", probeSvc, services)

	components, ok := c.subServices[probeSvc]
	if !ok {
		// probe service not registered yet
		components = []string{}
		c.subServices[probeSvc] = components
	}
	c.HealthServerStatusModifier.SetServingStatus(probeSvc, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// link subservices to the probe service
	for _, s := range services {
		c.subServiceToProbe[s] = probeSvc
		c.HealthServerStatusModifier.SetServingStatus(s, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}
	c.subServices[probeSvc] = append(components, services...)
}

// Check -
// Re-implements Check() function of grpc_health_v1.HealthServer interface
// in order to log if a probe service is about to fail.
func (c *Checker) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	//logrus.Debugf("Checker: Check service=%v", in.Service)

	switch in.Service {
	case Startup:
		fallthrough
	case Readiness:
		fallthrough
	case Liveness:
		resp, err := c.HealthServer.Check(ctx, in)
		if err == nil && resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
			svcInfo := ""
			// gather all related subservice' status to provide details on the failed probe
			if len(c.subServices[in.Service]) > 0 {
				svcInfo += `, subservices:[`
				for _, s := range c.subServices[in.Service] {
					svcInfo += fmt.Sprintf("%v:", s)
					status := `N/A,`
					if resp, err := c.HealthServer.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: s}); err == nil {
						status = fmt.Sprintf("%v,", resp.Status)
					}
					svcInfo += fmt.Sprintf("%v", status)
				}
				svcInfo += `]`
			}
			logrus.Infof(`Probe service "%v" [%v]%v`, in.Service, resp.Status, svcInfo)
		}
		return resp, err
	default:
		return c.HealthServer.Check(ctx, in)
	}
}

// Re-implements SetServingStatus function in order to evaluate probe service status
// in case a related subservice is subject to the call
func (c *Checker) SetServingStatus(service string, servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()

	//logrus.Debugf("Checker: Set serving status for service=%v, status=%v", service, servingStatus)
	switch service {
	case Startup:
		fallthrough
	case Readiness:
		fallthrough
	case Liveness:
		// in case of probe service only process serving status if no subservices
		if components, ok := c.subServices[service]; ok && len(components) > 0 {
			break
		}
		c.HealthServerStatusModifier.SetServingStatus(service, servingStatus)
	default:
		{
			c.HealthServerStatusModifier.SetServingStatus(service, servingStatus)
			// determine serving status of the related probe service if any
			if probeSvc, ok := c.subServiceToProbe[service]; ok {
				// belongs to a probe service (i.e. it's a subservice)
				if components, ok := c.subServices[probeSvc]; ok {
					// probe has subservices; aggregate serving status of components to get the probe status
					probeStatus := grpc_health_v1.HealthCheckResponse_NOT_SERVING
					if grpc_health_v1.HealthCheckResponse_SERVING == servingStatus {
						probeStatus = grpc_health_v1.HealthCheckResponse_SERVING
						for _, component := range components {
							// a subservice is not in SERVING status, the probe service must reflect it
							resp, err := c.HealthServer.Check(c.ctx, &grpc_health_v1.HealthCheckRequest{Service: component})
							if err != nil || (err == nil && resp.Status != grpc_health_v1.HealthCheckResponse_SERVING) {
								probeStatus = grpc_health_v1.HealthCheckResponse_NOT_SERVING
								break
							}
						}
					}
					//logrus.Debugf(`SetServingStatus: probe service "%v" [%v]`, probeSvc, probeStatus)
					c.HealthServerStatusModifier.SetServingStatus(probeSvc, probeStatus)
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
	return c.server.Serve(c.listener)
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
			return nil, fmt.Errorf("Failed to remove existing socket file: %v", err)
		}
		// create path if needed
		basePath := path.Dir(target)
		if _, err = os.Stat(basePath); os.IsNotExist(err) {
			if err = os.MkdirAll(basePath, os.ModePerm); err != nil {
				return nil, fmt.Errorf("Failed to create path for %v: %v", target, err)
			}
		}
	}
	// create listener for the URL
	lis, err := net.Listen(network, target)
	if err != nil {
		return nil, err
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
		subServices:                make(map[string][]string),
		subServiceToProbe:          make(map[string]string),
	}

	grpc_health_v1.RegisterHealthServer(s, checker)

	return checker, nil
}
