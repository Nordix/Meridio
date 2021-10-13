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
// 2 IPv4 and 2 IPv6 Static protocol sessions with BFD are present in the output.
// All 4 are known to the configuration. All sessions are up.
//
// After parsing:
// External connectivity must be OK.
// All sessions belonging to configured gateways must be up.
// Logs collected by the parser must contain all 4 sessions.
func TestParseProtocolsStaticWithBfd(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	log := ""
	configuredGatewayNamesByFamily = map[int]map[string]string{
		syscall.AF_INET:  {`NBR-gateway1`: `169.254.100.254`, `NBR-gateway3`: `169.254.100.253`},
		syscall.AF_INET6: {`NBR-gateway2`: `100:100::254`, `NBR-gateway4`: `100:100::253`},
	}
	bfdOutput := `
		BIRD 2.0.7 ready.
		NBR-BFD:
		IP address                Interface  State      Since         Interval  Timeout
		100:100::253              ext-vlan   Up         21:10:21.886    0.100    0.500
		169.254.100.253           ext-vlan   Up         21:10:21.886    0.100    0.500
		169.254.100.254           ext-vlan   Up         21:10:21.869    0.100    0.500
		100:100::254              ext-vlan   Up         21:10:21.869    0.100    0.500
	`
	expectedLog := `BIRD 2.0.7 ready.` + "\n" +
		`Name       Proto      Table      State  Since         Info` + "\n" +
		`NBR-gateway1 Static     master4    up     21:10:21.868 bfd: 169.254.100.254           ext-vlan   Up         21:10:21.869    0.100    0.500` + "\n" +
		`NBR-gateway2 Static     master6    up     21:10:21.868 bfd: 100:100::254              ext-vlan   Up         21:10:21.869    0.100    0.500` + "\n" +
		`NBR-gateway3 Static     master4    up     21:10:21.886 bfd: 169.254.100.253           ext-vlan   Up         21:10:21.886    0.100    0.500` + "\n" +
		`NBR-gateway4 Static     master6    up     21:10:21.886 bfd: 100:100::253              ext-vlan   Up         21:10:21.886    0.100    0.500` + "\n"

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
// 2 IPv4 and 2 IPv6 Static protocol sessions with BFD are present in the output.
// All 4 are known to the configuration. All Static sessions are up, but all IPv4
// BFD sessions are down.
//
// After parsing:
// External connectivity must be NOT OK.
// NOT all configured gateways are up.
// Logs collected by the parser must contain all 4 sessions.
func TestParseProtocolsStaticWithIPv4BfdDown(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	log := ""
	configuredGatewayNamesByFamily = map[int]map[string]string{
		syscall.AF_INET:  {`NBR-gateway1`: `169.254.100.254`, `NBR-gateway3`: `169.254.100.253`},
		syscall.AF_INET6: {`NBR-gateway2`: `100:100::254`, `NBR-gateway4`: `100:100::253`},
	}
	bfdOutput := `
    BIRD 2.0.7 ready.
    NBR-BFD:
    IP address                Interface  State      Since         Interval  Timeout
    100:100::253              ext-vlan   Up         21:10:21.886    0.100    0.500
    169.254.100.253           ext-vlan   Down       21:56:22.084    1.000    0.000
    169.254.100.254           ext-vlan   Down       21:56:22.040    1.000    0.000
    100:100::254              ext-vlan   Up         21:10:21.869    0.100    0.500
	`
	expectedLog := `BIRD 2.0.7 ready.` + "\n" +
		`Name       Proto      Table      State  Since         Info` + "\n" +
		`NBR-gateway1 Static     master4    up     21:10:21.868 bfd: 169.254.100.254           ext-vlan   Down       21:56:22.040    1.000    0.000` + "\n" +
		`NBR-gateway2 Static     master6    up     21:10:21.868 bfd: 100:100::254              ext-vlan   Up         21:10:21.869    0.100    0.500` + "\n" +
		`NBR-gateway3 Static     master4    up     21:10:21.886 bfd: 169.254.100.253           ext-vlan   Down       21:56:22.084    1.000    0.000` + "\n" +
		`NBR-gateway4 Static     master6    up     21:10:21.886 bfd: 100:100::253              ext-vlan   Up         21:10:21.886    0.100    0.500` + "\n"

	assert.NotNil(t, cs)
	assert.Empty(t, cs.Log())

	check(staticOutput, bfdOutput, cs, &log)

	t.Logf("cs: %v\n", cs)
	assert.True(t, cs.NoConnectivity())
	assert.True(t, cs.AnyGatewayDown())
	t.Logf("log:\n%v\n", log)
	assert.Equal(t, expectedLog, log)
}

// TestParseProtocolsStaticWithIPv4BfdDown -
// 2 IPv4 and 2 IPv6 Static protocol sessions in the output, all known to the configuration.
// Each has BFD configured except for 1 IPv6 gateway (NBR-gateway4).
// All Static sessions are up. The 1 IPv6 BFD session is down.
//
// After parsing:
// External connectivity must be OK.
// NOT all configured gateways are up.
// Logs collected by the parser must contain all 4 sessions.
func TestParseProtocolsStaticWithIPv6BfdDown(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	log := ""
	configuredGatewayNamesByFamily = map[int]map[string]string{
		syscall.AF_INET:  {`NBR-gateway1`: `169.254.100.254`, `NBR-gateway3`: `169.254.100.253`},
		syscall.AF_INET6: {`NBR-gateway2`: `100:100::254`, `NBR-gateway4`: `100:100::253`},
	}
	bfdOutput := `
		BIRD 2.0.7 ready.
		NBR-BFD:
		IP address                Interface  State      Since         Interval  Timeout
		169.254.100.253           ext-vlan   Up         22:13:19.439    0.100    0.500
		169.254.100.254           ext-vlan   Up         22:13:19.019    0.100    0.500
		100:100::254              ext-vlan   Down       22:13:30.771    1.000    0.000
	`
	expectedLog := `BIRD 2.0.7 ready.` + "\n" +
		`Name       Proto      Table      State  Since         Info` + "\n" +
		`NBR-gateway1 Static     master4    up     21:10:21.868 bfd: 169.254.100.254           ext-vlan   Up         22:13:19.019    0.100    0.500` + "\n" +
		`NBR-gateway2 Static     master6    up     21:10:21.868 bfd: 100:100::254              ext-vlan   Down       22:13:30.771    1.000    0.000` + "\n" +
		`NBR-gateway3 Static     master4    up     21:10:21.886 bfd: 169.254.100.253           ext-vlan   Up         22:13:19.439    0.100    0.500` + "\n" +
		`NBR-gateway4 Static     master6    up     21:10:21.886` + "\n"

	assert.NotNil(t, cs)
	assert.Empty(t, cs.Log())

	check(staticOutput, bfdOutput, cs, &log)

	t.Logf("cs: %v\n", cs)
	assert.False(t, cs.NoConnectivity())
	assert.True(t, cs.AnyGatewayDown())
	t.Logf("log:\n%v\n", log)
	assert.Equal(t, expectedLog, log)
}

var staticOutput string = `
BIRD 2.0.7 ready.
Name       Proto      Table      State  Since         Info
NBR-BFD    BFD        ---        up     21:10:09.901

NBR-gateway1 Static     master4    up     21:10:21.868
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     200
    Input filter:   default_rt
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

NBR-gateway2 Static     master6    up     21:10:21.868
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     200
    Input filter:   default_rt
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

NBR-gateway3 Static     master4    up     21:10:21.886
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     200
    Input filter:   default_rt
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

NBR-gateway4 Static     master6    up     21:10:21.886
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     200
    Input filter:   default_rt
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

`
