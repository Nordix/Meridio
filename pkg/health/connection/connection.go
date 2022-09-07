/*
Copyright (c) 2022 Nordix Foundation

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

package connection

import (
	"context"
	"fmt"
	"time"

	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type gRPCConnectionStateMonitor struct {
	healthService string
	*grpc.ClientConn
}

// Monitor -
// Monitors connection state changes and updates respective health service through context.
// Currently only gRPC connection is supported, which relies on EXPERIMENTAL API.
func Monitor(ctx context.Context, healthService string, cc interface{}) error {
	switch cc := cc.(type) {
	case *grpc.ClientConn:
		m := gRPCConnectionStateMonitor{
			healthService: healthService,
			ClientConn:    cc,
		}
		go func() {
			defer log.Logger.V(1).Info("Connection monitor exit", "service", m.healthService)
			for {
				s := m.GetState()
				health.SetServingStatus(ctx, m.healthService, s == connectivity.Ready)
				log.Logger.V(2).Info("Connection", "service", m.healthService, "state", s)

				// Note: gRPC will NOT establish underlying transport connection except for the
				// initial "dial" or unless the user tries to send sg and there's no backing connection.
				// Therefore trigger transport connect if gRPC connection state is Idle for 3 seconds,
				// to avoid the health service from getting stuck in NOT_SERVING if the user "remains silent"
				// for too long.
				waitCtx := ctx
				if s == connectivity.Idle {
					// TODO: configurable timeout
					timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
					waitCtx = timeoutCtx
					defer cancel()
				}
				// Block until connection state changes
				if !m.WaitForStateChange(waitCtx, s) {
					// context got timeout or canceled
					select {
					case <-ctx.Done():
						// main context done
						return
					default:
						// timeout; try re-connect
						m.Connect()
					}
				}
			}
		}()
		return nil
	default:
		return fmt.Errorf("unknown connection")
	}
}
