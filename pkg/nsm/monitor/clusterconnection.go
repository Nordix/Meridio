/*
Copyright (c) 2024 OpenInfra Foundation Europe

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

package monitor

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/edwarnicke/genericsync"
	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/nordix/meridio/pkg/log"
	"google.golang.org/grpc"
)

const defaultMonitorConnectTimeout time.Duration = 15 * time.Second

var monitorServers genericsync.Map[string, bool]

// ClusterConnectionMonitor -
// Based on NSM's cmd-dashboard-backend.
//
// Goal is to learn all NSMgr URLs via Forwarder NSE objects, and create a
// MonitorConnectionClient towards each, thus monitor connections of the whole
// cluster matching the MonitorScopeSelector.
//
// It is preferred to have a registryClient parameter that is connected to the local NSMgr:
// 1. The Find stream towards a local NSMgr seems to get closed after 15 (or 30?) seconds. Which is a rather
// favorable behaviour. That's because our Find() is merely watching Forwarder NSEs, yet we do intend to connect
// NSMgrs for Connection Monitoring purposes.
// Hence, if an NSMgr ConnectionMonitor stream terminates it's not self-explanatory that the Forwarder NSE should
// experience any change to trigger an event so that we could restore the ConnectionMonitor ASAP. (At least not
// after NSMgr container restart.)
// 2. No dependency towards the NSM kubernetes namespace is required.
//
// Note: Due to the nature of NSM's NSMgr Registry Server, if registryClient is connected to local NSMgr, then
// NSMgr localBypassNSEServer component will restore the endpoint's URL on Find(). That's because Find()'s main
// use case is to let Forwarders lookup NSEs upon NSC connection requests and connect them (either directly or
// via remote NSMgr depending on their location). The implication is, that Forwarder NSE local to our service
// can have its URL replaced with inode by NSMgr. Therefore, allow replacing the URL in such cases with preferably
// with the URL of the worker local NSMgr via checkMonitorServer function.
//
// (If registry client had a recvfd chain component, it would allow dialing and connecting the local Forwarder's
// Connection Monitor server. But since the aim is to connect to NSMgr for connection monitoring, this should
// be avoided. Parameter checkMonitorServer() can be used to replace URLs with "unix:///proc" prefix...)
func ClusterConnectionMonitor(
	ctx context.Context,
	registryClient registry.NetworkServiceEndpointRegistryClient,
	dialOptions []grpc.DialOption,
	selector *networkservice.MonitorScopeSelector,
	checkMonitorServer func(*url.URL) *url.URL,
	callback func(context.Context, *networkservice.ConnectionEvent),
) {
	logger := log.FromContextOrGlobal(ctx).WithValues("func", "ClusterConnectionMonitor")

	for ; ctx.Err() == nil; time.Sleep(time.Millisecond * 100) {
		var nseChannel = getNetworkServiceEndpointChannel(ctx, logger, registryClient)

	channelReadLoop:
		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return
			case nse, ok := <-nseChannel:
				if !ok {
					break channelReadLoop
				}

				monitorServerURL, err := url.Parse(nse.GetNetworkServiceEndpoint().Url)
				if err != nil {
					logger.Error(err, "Failed to parse raw server URL", "URL", nse.GetNetworkServiceEndpoint().Url)
					continue
				}

				// Note: A registry client that is connected to local NSMgr gets the URL of local endpoints
				// instead of the URL of the local NSMgr. As GetNetworkServiceEndpoint() looks up forwarder
				// endpoints, an "update" here would indicate forwarder related changes. Yet, the intension
				// is to create Connection Monitor streams towards NSMgrs. Hence, the URL shall be manually
				// overriden in such case.
				if checkMonitorServer != nil {
					monitorServerURL = checkMonitorServer(monitorServerURL)
				}

				monitorServerAddr := monitorServerURL.String()
				if _, exists := monitorServers.Load(monitorServerAddr); !exists {
					monitorServers.Store(monitorServerAddr, true)
					logger.V(1).Info("Extracted monitor server address", "addr", monitorServerAddr)

					// Start a goroutine for each monitoring session towards a monitor server (NSMgr)
					go func() {
						logger := logger.WithValues("address", monitorServerAddr)
						defer cleanup(logger, monitorServerAddr)
						conn, err := grpc.DialContext(
							ctx,
							func() string {
								switch monitorServerURL.Scheme {
								case "tcp":
									return monitorServerURL.Host
								}
								return monitorServerURL.String()
							}(),
							dialOptions...,
						)
						if err != nil {
							logger.Error(err, "Failed to dial server")
							return
						}

						streamCtx, streamCancel := context.WithCancel(ctx)
						defer streamCancel()
						clientConnections := networkservice.NewMonitorConnectionClient(conn)
						stream, err := createMonitorStream(streamCtx, clientConnections, selector)
						if err != nil {
							logger.Error(err, "Error from MonitorConnectionClient")
							return
						}
						logger.V(1).Info("Connected monitor server")

						for ctx.Err() == nil {
							event, err := stream.Recv()
							if err != nil {
								logger.Error(err, "Error from monitorConnection stream")
								// XXX: What if stream gets an error, yet the NSMgr is alive
								// or a NSMgr container restart occurred. The answer is that
								// propably the stream is not supposed to get suddenly closed
								// if NSMgr is alive.
								// On the other hand, our NSE channel merely monitors forwarder
								// NSEs that should not change upon above events.
								// However, seemingly the nse monitor stream along with the
								// endpoint channel get closed every 15 (30?) seconds in our case.
								// Which is beneficial, because at least after 15 (30?) seconds
								// delay the NSMgr MonitorConnection stream could get re-created.
								break
							}
							if callback != nil {
								callback(ctx, event)
							}
						}
					}()
				}
			}
		}
	}
}

// Lookup every forwarder NSE to extract NSMgr URL, that could be used to
// establish a connection towards each NSMgr and monitor connections.
// Since we are merely interested in NSMgrs, no need to check NSE state or
// lifetime.
func getNetworkServiceEndpointChannel(
	ctx context.Context, logger logr.Logger,
	registryClient registry.NetworkServiceEndpointRegistryClient,
) <-chan *registry.NetworkServiceEndpointResponse {
	streamNse, err := registryClient.Find(ctx, &registry.NetworkServiceEndpointQuery{
		Watch: true,
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceNames: []string{"forwarder"},
		},
	})
	if err != nil {
		logger.Error(err, "Failed to perform Find NSE request")
	}

	return readNetworkServiceEndpointChannel(streamNse, logger)
}

func cleanup(logger logr.Logger, monitorServerAddr string) {
	logger.V(1).Info("Cleanup", "addr", monitorServerAddr)
	monitorServers.Delete(monitorServerAddr)
}

func readNetworkServiceEndpointChannel(
	stream registry.NetworkServiceEndpointRegistry_FindClient,
	logger logr.Logger,
) <-chan *registry.NetworkServiceEndpointResponse {
	result := make(chan *registry.NetworkServiceEndpointResponse)
	go func() {
		defer func() {
			close(result)
		}()
		for msg, err := stream.Recv(); err == nil; msg, err = stream.Recv() {
			select {
			case result <- msg:
				continue
			case <-stream.Context().Done():
				logger.V(1).Info("context closed")
				return
			}
		}
	}()
	return result
}

// createMonitorStream -
// Attempts to create the monitor stream within defaultMonitorConnectTimeout.
// Expects a cancellable context to be passed in.
// The caller must call cancel() if an error is returned to avoid context leaks.
//
// Rationale:
// We cannot be certain that DialContext is configured via dial options to block
// until a connection is established. Therefore, this function ensures the monitor
// RPC call does not get stuck indefinitely if the server disappears before a
// connection is established or if the call cannot return in a timely manner.
func createMonitorStream(
	ctx context.Context,
	clientConnections networkservice.MonitorConnectionClient,
	selector *networkservice.MonitorScopeSelector,
) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	var stream networkservice.MonitorConnection_MonitorConnectionsClient
	var err error
	// Create a context with a timeout for the stream creation attempt
	dialTimeoutCtx, cancel := context.WithTimeout(ctx, defaultMonitorConnectTimeout)
	defer cancel()

	// Try to create the stream
	done := make(chan struct{})
	go func() {
		defer close(done)
		stream, err = clientConnections.MonitorConnections(ctx, selector)
	}()

	select {
	case <-done:
		// Stream creation returned (success or failure)
		if err != nil {
			// Error returned during stream creation
			return nil, fmt.Errorf("failed to create monitor stream: %w", err)
		}
		// Success
		return stream, err
	case <-dialTimeoutCtx.Done():
		// Timeout reached while attempting to create the stream
		// Cancel the RPC context to clean up any ongoing RPC and avoid goroutine leak
		return nil, fmt.Errorf("failed to create stream within timeout: %w", dialTimeoutCtx.Err())
	}
}
