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
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestStatusNoConfig(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	assert := assert.New(t)
	cs := connectivity.NewConnectivityStatus()
	assert.NotNil(cs.StatusMap())
	assert.NotNil(cs.Logp())
	cs.SetNoConfig(syscall.AF_INET)
	cs.SetNoConfig(syscall.AF_INET6)

	t.Logf("cs: %v\n", cs.String())
	assert.Equal(connectivity.NoConfig, cs.Status())
	assert.False(cs.AnyGatewayDown())
	assert.True(cs.NoConnectivity())
	assert.Empty(cs.Log())
	assert.Empty(cs.StatusMap())
}

func TestStatusNoIPv4Config(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	assert := assert.New(t)
	cs := connectivity.NewConnectivityStatus()
	assert.NotNil(cs.StatusMap())
	assert.NotNil(cs.Logp())

	cs.SetNoConfig(syscall.AF_INET)
	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)
	cs.SetGatewayDown("gateway-4")

	assert.True(cs.AnyGatewayDown())
	assert.False(cs.NoConnectivity())
	assert.Len(cs.StatusMap(), 2)
}

func TestStatusNoIPv6Config(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	assert := assert.New(t)
	cs := connectivity.NewConnectivityStatus()
	assert.NotNil(cs.StatusMap())
	assert.NotNil(cs.Logp())

	cs.SetNoConfig(syscall.AF_INET6)
	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayUp("gateway-2", syscall.AF_INET)

	assert.False(cs.AnyGatewayDown())
	assert.False(cs.NoConnectivity())
	assert.Len(cs.StatusMap(), 2)
}

func TestStatusNoIPv4Conn(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	assert := assert.New(t)
	cs := connectivity.NewConnectivityStatus()
	assert.NotNil(cs.StatusMap())
	assert.NotNil(cs.Logp())

	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)

	assert.False(cs.AnyGatewayDown())
	assert.True(cs.NoConnectivity())
	assert.Len(cs.StatusMap(), 1)
}

func TestStatusNoIPv4ConnWithGatewayDown(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	assert := assert.New(t)
	cs := connectivity.NewConnectivityStatus()
	assert.NotNil(cs.StatusMap())
	assert.NotNil(cs.Logp())

	cs.SetGatewayDown("gateway-1")
	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)

	assert.True(cs.AnyGatewayDown())
	assert.True(cs.NoConnectivity())
	assert.Len(cs.StatusMap(), 2)
}

func TestStatusNoIPv6Conn(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	assert := assert.New(t)
	cs := connectivity.NewConnectivityStatus()
	assert.NotNil(cs.StatusMap())
	assert.NotNil(cs.Logp())

	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayUp("gateway-2", syscall.AF_INET)

	assert.False(cs.AnyGatewayDown())
	assert.True(cs.NoConnectivity())
	assert.Len(cs.StatusMap(), 2)
}

func TestStatusNoIPv6ConnWithGatewayDown(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	assert := assert.New(t)
	cs := connectivity.NewConnectivityStatus()
	assert.NotNil(cs.StatusMap())
	assert.NotNil(cs.Logp())

	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayUp("gateway-2", syscall.AF_INET)
	cs.SetGatewayDown("gateway-3")

	assert.True(cs.AnyGatewayDown())
	assert.True(cs.NoConnectivity())
	assert.Len(cs.StatusMap(), 3)
}

func TestStatusDualStack(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	assert := assert.New(t)
	cs := connectivity.NewConnectivityStatus()
	assert.NotNil(cs.StatusMap())
	assert.NotNil(cs.Logp())

	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayUp("gateway-2", syscall.AF_INET)
	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)
	cs.SetGatewayUp("gateway-4", syscall.AF_INET6)

	assert.False(cs.AnyGatewayDown())
	assert.False(cs.NoConnectivity())
	assert.Len(cs.StatusMap(), 4)
}

func TestStatusDualStackWithGatewayDown(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	assert := assert.New(t)
	cs := connectivity.NewConnectivityStatus()
	assert.NotNil(cs.StatusMap())
	assert.NotNil(cs.Logp())

	cs.SetGatewayUp("gateway-1", syscall.AF_INET)
	cs.SetGatewayDown("gateway-2")
	cs.SetGatewayUp("gateway-3", syscall.AF_INET6)
	cs.SetGatewayDown("gateway-4")

	assert.True(cs.AnyGatewayDown())
	assert.False(cs.NoConnectivity())
	assert.Len(cs.StatusMap(), 4)
}
