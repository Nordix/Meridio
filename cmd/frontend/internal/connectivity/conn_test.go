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

package connectivity_test

import (
	"syscall"
	"testing"

	"github.com/nordix/meridio/cmd/frontend/internal/connectivity"
	"go.uber.org/goleak"
	"gotest.tools/assert"
)

func TestStatusNoConfig(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	assert.Assert(t, cs.StatusMap() != nil)
	assert.Assert(t, cs.Logp() != nil)
	cs.SetNoConfig(syscall.AF_INET)
	cs.SetNoConfig(syscall.AF_INET6)

	t.Logf("cs: %v\n", cs.String())
	assert.Equal(t, connectivity.NoConfig, cs.Status())
	assert.Equal(t, false, cs.AnyGatewayDown())
	assert.Equal(t, true, cs.NoConnectivity())
	assert.Equal(t, "", cs.Log())
	assert.Equal(t, 0, len(cs.StatusMap()))
}

func TestStatusNoIPv4Config(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	assert.Assert(t, cs.StatusMap() != nil)
	assert.Assert(t, cs.Logp() != nil)

	cs.SetNoConfig(syscall.AF_INET)
	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)
	cs.SetGatewayDown("gateway-4")

	assert.Equal(t, true, cs.AnyGatewayDown())
	assert.Equal(t, false, cs.NoConnectivity())
	assert.Equal(t, 2, len(cs.StatusMap()))
}

func TestStatusNoIPv6Config(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	assert.Assert(t, cs.StatusMap() != nil)
	assert.Assert(t, cs.Logp() != nil)

	cs.SetNoConfig(syscall.AF_INET6)
	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayUp("gateway-2", syscall.AF_INET)

	assert.Equal(t, false, cs.AnyGatewayDown())
	assert.Equal(t, false, cs.NoConnectivity())
	assert.Equal(t, 2, len(cs.StatusMap()))
}

func TestStatusNoIPv4Conn(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	assert.Assert(t, cs.StatusMap() != nil)
	assert.Assert(t, cs.Logp() != nil)

	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)

	assert.Equal(t, false, cs.AnyGatewayDown())
	assert.Equal(t, true, cs.NoConnectivity())
	assert.Equal(t, 1, len(cs.StatusMap()))
}

func TestStatusNoIPv4ConnWithGatewayDown(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	assert.Assert(t, cs.StatusMap() != nil)
	assert.Assert(t, cs.Logp() != nil)

	cs.SetGatewayDown("gateway-1")
	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)

	assert.Equal(t, true, cs.AnyGatewayDown())
	assert.Equal(t, true, cs.NoConnectivity())
	assert.Equal(t, 2, len(cs.StatusMap()))
}

func TestStatusNoIPv6Conn(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	assert.Assert(t, cs.StatusMap() != nil)
	assert.Assert(t, cs.Logp() != nil)

	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayUp("gateway-2", syscall.AF_INET)

	assert.Equal(t, false, cs.AnyGatewayDown())
	assert.Equal(t, true, cs.NoConnectivity())
	assert.Equal(t, 2, len(cs.StatusMap()))
}

func TestStatusNoIPv6ConnWithGatewayDown(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	assert.Assert(t, cs.StatusMap() != nil)
	assert.Assert(t, cs.Logp() != nil)

	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayUp("gateway-2", syscall.AF_INET)
	cs.SetGatewayDown("gateway-3")

	assert.Equal(t, true, cs.AnyGatewayDown())
	assert.Equal(t, true, cs.NoConnectivity())
	assert.Equal(t, 3, len(cs.StatusMap()))
}

func TestStatusDualStack(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	assert.Assert(t, cs.StatusMap() != nil)
	assert.Assert(t, cs.Logp() != nil)

	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayUp("gateway-2", syscall.AF_INET)
	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)
	cs.SetGatewayUp("gateway-4", syscall.AF_INET6)

	assert.Equal(t, false, cs.AnyGatewayDown())
	assert.Equal(t, false, cs.NoConnectivity())
	assert.Equal(t, 4, len(cs.StatusMap()))
}

func TestStatusDualStackWithGatewayDown(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	assert.Assert(t, cs.StatusMap() != nil)
	assert.Assert(t, cs.Logp() != nil)

	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayDown("gateway-2")
	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)
	cs.SetGatewayDown("gateway-4")

	assert.Equal(t, true, cs.AnyGatewayDown())
	assert.Equal(t, false, cs.NoConnectivity())
	assert.Equal(t, 4, len(cs.StatusMap()))
}
