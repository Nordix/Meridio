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

package bird_test

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/nordix/meridio/cmd/frontend/internal/connectivity"
)

// TestParseProtocolsStaticWithBfd -
// 1 IPv4 and 1 IPv6 Static protocol sessions with BFD are present in the output.
// All are known to the configuration. All sessions are up.
//
// After parsing:
// External connectivity must be OK.
// All sessions belonging to configured gateways must be up.
// Logs collected by the parser must contain all sessions.
func TestParseProtocolsStaticWithBfd(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	log := ""
	configuredGatewayNamesByFamily = map[string]*testGateway{
		`NBR-gateway1`: {ip: `169.254.100.150`, af: syscall.AF_INET, bfd: true},
		`NBR-gateway2`: {ip: `100:100::150`, af: syscall.AF_INET6, bfd: true},
	}
	bfdOutput := `
		BIRD 3.1.4 ready.
		NBR-BFD:
		IP address                Interface  State      Since         Interval  Timeout
		169.254.100.150           eth0.100   Up         08:46:47.676    0.300    1.500
		100:100::150              eth0.100   Up         08:46:47.676    0.300    1.500
	`
	expectedLog := `BIRD 3.1.4 ready.` + "\n" +
		`Name       Proto      Table      State  Since         Info` + "\n" +
		`NBR-gateway1 Static     master4    up     08:46:47.676   bfd: 169.254.100.150           eth0.100   Up         08:46:47.676    0.300    1.500` + "\n" +
		`NBR-gateway2 Static     master6    up     08:46:47.767   bfd: 100:100::150              eth0.100   Up         08:46:47.676    0.300    1.500` + "\n"

	assert.NotNil(t, cs)
	assert.Empty(t, cs.Log())

	check(staticOutput, bfdOutput, cs, &log)

	t.Logf("cs: %v\n", cs)
	assert.False(t, cs.NoConnectivity())
	assert.False(t, cs.AnyGatewayDown())
	t.Logf("log:\n%v\n", log)
	assert.Equal(t, expectedLog, log)
}

// TestParseProtocolsStaticWithIPv4BfdDown -
// 1 IPv4 and 1 IPv6 Static protocol sessions with BFD are present in the output.
// All are known to the configuration. All Static sessions are up, but all IPv4
// BFD sessions are down.
//
// After parsing:
// External connectivity must be NOT OK.
// NOT all configured gateways are up.
// Logs collected by the parser must contain all sessions.
func TestParseProtocolsStaticWithIPv4BfdDown(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	log := ""
	configuredGatewayNamesByFamily = map[string]*testGateway{
		`NBR-gateway1`: {ip: `169.254.100.150`, af: syscall.AF_INET, bfd: true},
		`NBR-gateway2`: {ip: `100:100::150`, af: syscall.AF_INET6, bfd: true},
	}
	bfdOutput := `
	    BIRD 3.1.4 ready.
		NBR-BFD:
		IP address                Interface  State      Since         Interval  Timeout
		169.254.100.150           eth0.100   Down       08:46:47.676    0.300    1.500
		100:100::150              eth0.100   Up         08:46:47.676    0.300    1.500
	`
	expectedLog := `BIRD 3.1.4 ready.` + "\n" +
		`Name       Proto      Table      State  Since         Info` + "\n" +
		`NBR-gateway1 Static     master4    up     08:46:47.676   bfd: 169.254.100.150           eth0.100   Down       08:46:47.676    0.300    1.500` + "\n" +
		`NBR-gateway2 Static     master6    up     08:46:47.767   bfd: 100:100::150              eth0.100   Up         08:46:47.676    0.300    1.500` + "\n"
	assert.NotNil(t, cs)
	assert.Empty(t, cs.Log())

	check(staticOutput, bfdOutput, cs, &log)

	t.Logf("cs: %v\n", cs)
	assert.True(t, cs.NoConnectivity())
	assert.True(t, cs.AnyGatewayDown())
	t.Logf("log:\n%v\n", log)
	assert.Equal(t, expectedLog, log)
}

// TestParseProtocolsStaticWitLingeringBfd -
// 1 IPv4 and 1 IPv6 Static protocol sessions in the output, all known to the configuration.
// IPv4 Static protocols have BFD configured, while IPv6 have none. However independent IPv6
// BFD sessions do exist for each IPv6 gateway IP.
// All Static sessions are up. The IPv4 BFD sessions are up. The 2 independent IPv6 BFD sessions are down.
//
// After parsing:
// External connectivity must be OK.
// All configured gateways must be up.
// Logs collected by the parser must contain all sessions.
func TestParseProtocolsStaticWithLingeringBfd(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	log := ""
	configuredGatewayNamesByFamily = map[string]*testGateway{
		`NBR-gateway1`: {ip: `169.254.100.150`, af: syscall.AF_INET, bfd: true},
		`NBR-gateway2`: {ip: `100:100::150`, af: syscall.AF_INET6, bfd: false},
	}
	bfdOutput := `
		BIRD 3.1.4 ready.
		NBR-BFD:
		IP address                Interface  State      Since         Interval  Timeout
		169.254.100.150           eth0.100   Up         08:46:47.676    0.300    1.500
		100:100::150              eth0.100   Down       08:46:47.676    0.300    1.500

	`
	expectedLog := `BIRD 3.1.4 ready.` + "\n" +
		`Name       Proto      Table      State  Since         Info` + "\n" +
		`NBR-gateway1 Static     master4    up     08:46:47.676   bfd: 169.254.100.150           eth0.100   Up         08:46:47.676    0.300    1.500` + "\n" +
		`NBR-gateway2 Static     master6    up     08:46:47.767  ` + "\n"

	assert.NotNil(t, cs)
	assert.Empty(t, cs.Log())

	check(staticOutput, bfdOutput, cs, &log)

	t.Logf("cs: %v\n", cs)
	assert.False(t, cs.NoConnectivity())
	assert.False(t, cs.AnyGatewayDown())
	t.Logf("log:\n%v\n", log)
	assert.Equal(t, expectedLog, log)
}

// TestParseProtocolsStaticWithMissingBfd -
// 1 IPv4 and 1 IPv6 Static protocol sessions in the output, all known to the configuration.
// All have BFD configured.
// All Static sessions are up. IPv4 BFD sessions are up, however IPv6 BFD sessions are missing.
//
// After parsing:
// External connectivity must be NOT OK.
// NOT all configured gateways are up.
// Logs collected by the parser must contain all sessions.
func TestParseProtocolsStaticWithMissingBfd(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	log := ""
	configuredGatewayNamesByFamily = map[string]*testGateway{
		`NBR-gateway1`: {ip: `169.254.100.150`, af: syscall.AF_INET, bfd: true},
		`NBR-gateway2`: {ip: `100:100::150`, af: syscall.AF_INET6, bfd: true},
	}

	bfdOutput := `
		BIRD 3.1.4 ready.
		NBR-BFD:
		IP address                Interface  State      Since         Interval  Timeout
		169.254.100.150           eth0.100   Up         08:46:47.676    0.300    1.500
	`
	expectedLog := `BIRD 3.1.4 ready.` + "\n" +
		`Name       Proto      Table      State  Since         Info` + "\n" +
		`NBR-gateway1 Static     master4    up     08:46:47.676   bfd: 169.254.100.150           eth0.100   Up         08:46:47.676    0.300    1.500` + "\n" +
		`NBR-gateway2 Static     master6    up     08:46:47.767   bfd: no session` + "\n"

	assert.NotNil(t, cs)
	assert.Empty(t, cs.Log())

	check(staticOutput, bfdOutput, cs, &log)

	t.Logf("cs: %v\n", cs)
	assert.True(t, cs.NoConnectivity())
	assert.True(t, cs.AnyGatewayDown())
	t.Logf("log:\n%v\n", log)
	assert.Equal(t, expectedLog, log)
}

var staticOutput string = `
BIRD 3.1.4 ready.
Name       Proto      Table      State  Since         Info
NBR-gateway1 Static     master4    up     08:46:47.676  
  Created:            08:46:47.676
  Channel ipv4
    State:          UP
    Import state:   UP
    Export state:   DOWN
    Table:          master4
    Preference:     200
    Input filter:   default_rt
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   RX limit      limit   accepted
      Import updates:              2          0          0          0          0          0          2
      Import withdraws:            1          0        ---          0        ---        ---          1
      Export updates:              0          0          0          0        ---          0          0
      Export withdraws:            0        ---        ---          0        ---        ---          0

NBR-gateway2 Static     master6    up     08:46:47.767  
  Created:            08:46:47.676
  Channel ipv6
    State:          UP
    Import state:   UP
    Export state:   DOWN
    Table:          master6
    Preference:     200
    Input filter:   default_rt
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   RX limit      limit   accepted
      Import updates:              2          0          0          0          0          0          2
      Import withdraws:            1          0        ---          0        ---        ---          1
      Export updates:              0          0          0          0        ---          0          0
      Export withdraws:            0        ---        ---          0        ---        ---          0

NBR-BFD    BFD        ---        up     08:46:38.612  
  Created:            08:46:38.612
`
