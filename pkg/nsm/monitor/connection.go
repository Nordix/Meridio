/*
Copyright (c) 2023 Nordix Foundation
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
	"io"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConnectionMonitor -
// ConnectionMonitor monitors NSM connection events to keep track of and log
// important changes. Currently, monitors connections this proxy is part of.
// TODO: make more generic
func ConnectionMonitor(ctx context.Context, name string, monitorConnectionClient networkservice.MonitorConnectionClient) {
	if monitorConnectionClient == nil {
		return
	}

	logger := log.Logger.WithValues("func", "ConnectionMonitor", "name", name)
	logger.V(1).Info("Start NSM connection monitor")
	defer logger.V(1).Info("Stopped NSM connection monitor")
	monitorScope := &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{
			{
				Name: name, // interested in connections whose path includes name
			},
		},
	}

	_ = retry.Do(func() error {
		monitorConnectionsClient, err := monitorConnectionClient.MonitorConnections(ctx, monitorScope)
		if err != nil {
			return fmt.Errorf("failed to create connection monitor client: %w", err)
		}
		for {
			mccResponse, err := monitorConnectionsClient.Recv()
			if err != nil {
				if status.Code(err) != codes.Canceled {
					// not shutdown caused cancellation
					logger.Info("Connection monitor lost contact with local NSMgr")
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("connection monitor client receive error: %w", err)
			}
			for _, connection := range mccResponse.Connections {
				if connection.GetPath() != nil && len(connection.GetPath().GetPathSegments()) >= 1 {
					index := -1 // indicates at which path segment index the name is located
					segmentNum := len(connection.GetPath().GetPathSegments())
					pathSegments := make([]*networkservice.PathSegment, segmentNum)
					// double check the name is involed and build temp pathSegments
					// with reduced information for logging
					for i, s := range connection.GetPath().GetPathSegments() {
						if s.Name == name {
							index = i // found the name
						}
						pathSegments[i] = &networkservice.PathSegment{Name: s.Name, Id: s.Id}
					}
					if index < 0 {
						continue
					}

					logger := logger.WithValues(
						"event type", mccResponse.Type,
						"connection state", connection.GetState(),
						"connContext", connection.Context,
						"pathSegments", pathSegments,
					)
					if connection.GetMechanism() != nil && connection.GetMechanism().GetParameters() != nil {
						logger = logger.WithValues("mechParameters", connection.GetMechanism().GetParameters())
					}

					if connection.GetState() == networkservice.State_DOWN || mccResponse.Type == networkservice.ConnectionEventType_DELETE {
						msg := "Connection monitor received delete event" // connection closed (e.g. due to NSM heal with reselect)
						if connection.GetState() == networkservice.State_DOWN {
							msg = "Connection monitor received down event" // control plane is down
						}
						logger.Info(msg)
					}
				}
			}
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(5*time.Second),
		retry.WithErrorIngnored())

}
