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

package ipcontext

import (
	"context"
	"time"

	"github.com/nordix/meridio/pkg/networking"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type ipcontextClient struct {
	ics            ipContextSetter
	ipReleaseDelay time.Duration
}

// NewClient
func NewClient(ipContextSetter ipContextSetter, ipReleaseDelay time.Duration) networkservice.NetworkServiceClient {
	return &ipcontextClient{
		ics:            ipContextSetter,
		ipReleaseDelay: ipReleaseDelay,
	}
}

// Note: It's really hard to asses when allocated IPs are no longer needed,
// without risking leasing them early too another connection

// Request
// Note: Currently IPs are allowed to get released here to avoid leaking IPs
// for connections that did not get established and were abandoned.
// Unfortunately, it's hard to tell apart non-established connections and
// connections subject to NSM heal. (Probably path entries in the request
// could serve as an indication regarding that though.) Ultimately, we want to
// avoid releasing IPs for which a connection might still exist "within NSM"
// (and thus so could associated interfaces in PODs) like in case of NSM heal.
//
// For example, a Close() call is expected if Request fails during NSM heal.
// If said Close() would block for 15 seconds, releasing IPs should be avoided
// before a chance could have been given to a "reconnect" Request() to cancel
// the delayed release (and keep the IPs associated with the connection).
// In theory, this could be achieved e.g. by re-acquiring the IPs at the start of
// Close() via SetIPContext(). But update of IPContext must be avoided for Close().
// (Remember, that if IPs in IPContext get updated during NSM heal during Request()
// but said request fails, then Close() will still include the old IPContext.)
// Therefore, let's go with a slightly longer delay instead for release, so that it
// could be cancelled in time if needed. The delay of the release should exceed the
// request timeout.
func (icc *ipcontextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	err := icc.ics.SetIPContext(ctx, request.Connection, networking.NSC)
	if err != nil {
		if request.Connection.GetMechanism() == nil {
			// no established connection, do not risk leaking IPs in IPAM (e.g. in case client gives up)
			// Note: Mechanism can be nil during NSM heal as well. Yet, the old interface with the
			// (old) addresses might be left intact if initial Close as part of reconnect failed.
			_ = icc.ics.UnsetIPContext(context.Background(), request.Connection, networking.NSC, icc.ipReleaseDelay)
		}
		return nil, err
	}
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		if request.Connection.GetMechanism() == nil {
			// no established connection, do not risk leaking IPs in IPAM (e.g. in case client gives up)
			// Note: Mechanism can be nil during NSM heal as well. Yet, the old interface with the
			// (old) addresses might be left intact if initial Close as part of reconnect failed.
			// Note: Although the proxy won't try to recover connections via NSM Monitor Connection,
			// but Mechanism could be recovered that way to tell apart non-established conns and conns
			// under heal.
			_ = icc.ics.UnsetIPContext(context.Background(), request.Connection, networking.NSC, icc.ipReleaseDelay)
		}
		return nil, err
	}

	return conn, nil
}

// Close
// XXX: Planned to release IPs without delay if Close() was successful
// (by allowing delayed IP release to get canceled). Turned out, that on
// recent NSM versions (e.g. v1.13.0) a restarted nsmgr will reply a Close()
// with no error if its beginServer does not recognise the connection. Thus,
// despite no error old interfaces are not removed (~Close() is ambigous).
func (icc *ipcontextClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	err := icc.ics.UnsetIPContext(ctx, conn, networking.NSC, icc.ipReleaseDelay)
	if err != nil {
		return nil, err
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
