/*
Copyright (c) 2026 OpenInfra Foundation Europe

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

package nsp_test

import (
	"context"
	"errors"
	"testing"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp"
	memory "github.com/nordix/meridio/pkg/nsp/registry/memory"
	"github.com/nordix/meridio/pkg/nsp/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeWatchServer is the server side of a Watch stream. Send calls onSend if
// set, then returns sendErr; the other grpc.ServerStream methods are not used
// by the server.
type fakeWatchServer struct {
	grpc.ServerStream
	ctx     context.Context
	onSend  func()
	sendErr error
}

func (f *fakeWatchServer) Send(*nspAPI.TargetResponse) error {
	if f.onSend != nil {
		f.onSend()
	}
	return f.sendErr
}

func (f *fakeWatchServer) Context() context.Context { return f.ctx }

// TestServer_Watch_ReturnsSendError verifies that a failed Send ends the Watch
// call with that error, keeping its gRPC status code, instead of with OK.
func TestServer_Watch_ReturnsSendError(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := nsp.NewServer(memory.New())

	// The registry delivers an initial target list, so the server sends at
	// once, and the send fails.
	stream := &fakeWatchServer{
		ctx:     context.Background(),
		sendErr: status.Error(codes.Unavailable, "transport closing"),
	}
	err := server.Watch(&nspAPI.Target{}, stream)

	assert.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err), "status code of the Send error must be kept")
}

// TestServer_Watch_ClientGone verifies that a Watch whose client went away
// ends without error.
func TestServer_Watch_ClientGone(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := nsp.NewServer(memory.New())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the client is already gone
	stream := &fakeWatchServer{ctx: ctx}

	assert.NoError(t, server.Watch(&nspAPI.Target{}, stream))
}

// TestServer_Watch_ClientGoneDuringSend verifies that a Send failing because
// the client went away ends the Watch without error, like a client that went
// away between sends.
func TestServer_Watch_ClientGoneDuringSend(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := nsp.NewServer(memory.New())

	// The client is still there when the server starts sending the initial
	// target list, so the server takes the send path. The client goes away
	// during the send (onSend cancels the context), and the send fails.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := &fakeWatchServer{
		ctx:     ctx,
		onSend:  cancel,
		sendErr: status.Error(codes.Canceled, "context canceled"),
	}

	assert.NoError(t, server.Watch(&nspAPI.Target{}, stream))
}

// watchRecorder is a target registry whose Watch records the context it gets
// and fails, ending the server's Watch at once.
type watchRecorder struct {
	types.TargetRegistry
	ctx context.Context
}

func (r *watchRecorder) Watch(ctx context.Context, _ *nspAPI.Target) (types.TargetWatcher, error) {
	r.ctx = ctx
	return nil, errors.New("recorded")
}

// TestServer_Watch_PassesStreamContext verifies that the server passes the
// stream's context to the registry, so the registry can skip a Watch whose
// client already gave up.
func TestServer_Watch_PassesStreamContext(t *testing.T) {
	registry := &watchRecorder{}
	server := nsp.NewServer(registry)
	// A cancellable context is a unique object, so the registry can only have
	// received it from the stream.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := &fakeWatchServer{ctx: ctx}

	assert.Error(t, server.Watch(&nspAPI.Target{}, stream))
	assert.Equal(t, stream.ctx, registry.ctx)
}
