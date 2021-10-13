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

	"github.com/nordix/meridio/cmd/frontend/internal/bird"
	"github.com/nordix/meridio/cmd/frontend/internal/connectivity"
)

var extInterface string = `ext-vlan`

var configuredGatewayNamesByFamily map[int]map[string]string = map[int]map[string]string{
	syscall.AF_INET:  {},
	syscall.AF_INET6: {},
}

func getGatewayIPByName(name string) (string, int, bool) {
	ok := false
	addr := ""
	family := syscall.AF_UNSPEC

	if addr, ok = configuredGatewayNamesByFamily[syscall.AF_INET][name]; ok {
		family = syscall.AF_INET
	} else if addr, ok = configuredGatewayNamesByFamily[syscall.AF_INET6][name]; ok {
		family = syscall.AF_INET6
	}

	return addr, family, ok
}

// parse protocol outputs to determine connectivity
func check(protocolOutput, bfdOutput string, cs *connectivity.ConnectivityStatus, logp *string) {
	bird.ParseProtocols(protocolOutput, logp, func(name string, options ...bird.Option) {
		ok := false
		ip := ""
		var family int
		if ip, family, ok = getGatewayIPByName(name); !ok {
			// no configured gateway found for the name
			return
		}

		// extend protocol options with external inteface, gateway ip, bfd sessions
		opts := append([]bird.Option{
			bird.WithInterface(extInterface),
			bird.WithNeighbor(ip),
			bird.WithBfdSessions(bfdOutput),
		}, options...)

		p := bird.NewProtocol(opts...)
		// check if protocol session is down
		if bird.ProtocolDown(p) {
			cs.SetGatewayDown(name) // neighbor protocol down
		} else {
			cs.SetGatewayUp(name, family) // neighbor protocol up
		}
	})
}

// TestParseProtocols -
// 2 IPv4 and 2 IPv6 BGP protocol sessions are present in the output. All the sessions are up.
// Only 1 IPv4 and 1 IPv6 sessions are known to the configuration. BFD is not configured.
//
// After parsing:
// External connectivity must be OK.
// All BGP sessions known to configuration must be up.
// Logs collected by the parser must contain all 4 BGP sessions.
func TestParseProtocols(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	configuredGatewayNamesByFamily = map[int]map[string]string{
		syscall.AF_INET:  {`NBR-gateway3`: `169.254.100.253`},
		syscall.AF_INET6: {`NBR-gateway4`: `100:100::253`},
	}
	expectedLog := `BIRD 2.0.7 ready.` + "\n" +
		`Name       Proto      Table      State  Since         Info` + "\n" +
		`NBR-gateway1 BGP        ---        up     16:03:22.388  Established    Neighbor address: 169.254.100.254%ext-vlan` + "\n" +
		`NBR-gateway2 BGP        ---        up     17:53:39.604  Established    Neighbor address: 100:100::254%ext-vlan` + "\n" +
		`NBR-gateway3 BGP        ---        up     17:18:30.468  Established    Neighbor address: 169.254.100.253%ext-vlan` + "\n" +
		`NBR-gateway4 BGP        ---        up     17:53:40.211  Established    Neighbor address: 100:100::253%ext-vlan` + "\n"

	assert.NotNil(t, cs)
	assert.Equal(t, "", cs.Log())

	check(bgpOutput, "", cs, cs.Logp())

	t.Logf("cs: %v\n", cs)
	assert.False(t, cs.NoConnectivity())
	assert.False(t, cs.AnyGatewayDown())
	assert.Equal(t, expectedLog, cs.Log())
}

// TestParseProtocolsWithBfd -
// 2 IPv4 and 2 IPv6 (link-local addr) BGP protocol sessions are present in the output.
// Only 1 IPv4 and 1 IPv6 sessions are known to the configuration (configuredGatewayNamesByFamily).
// The IPv4 session not known to the config (NBR-gateway3) is down.
// BFD is configured for all the neighbors, and all are up.
//
// After parsing:
// External connectivity must be OK.
// All BGP sessions part of the configuration must be up.
func TestParseProtocolsWithBfd(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	configuredGatewayNamesByFamily = map[int]map[string]string{
		syscall.AF_INET:  {`NBR-gateway1`: `169.254.100.254`},
		syscall.AF_INET6: {`NBR-gateway2`: `fe80::beef`},
	}
	bfdOutput := `
		BIRD 2.0.7 ready.
		NBR-BFD:
		IP address                Interface  State      Since         Interval  Timeout
		fe80::beef                ext-vlan   Up         16:03:25.387    0.100    0.500
		fe80::beee                ext-vlan   Up         16:03:25.943    0.100    0.500
		169.254.100.253           ext-vlan   Up         16:03:18.349    0.100    0.500
		169.254.100.254           ext-vlan   Up         16:03:18.349    0.100    0.500
	`

	assert.NotNil(t, cs)
	assert.Empty(t, cs.Log())

	check(bgpLinkLocalOutput, bfdOutput, cs, nil)

	t.Logf("cs: %v\n", cs)
	assert.False(t, cs.NoConnectivity())
	assert.False(t, cs.AnyGatewayDown())
}

var bgpLinkLocalOutput string = `
BIRD 2.0.7 ready.
Name       Proto      Table      State  Since         Info
NBR-BFD    BFD        ---        up     16:03:17.278  

NBR-gateway1 BGP        ---        up     16:03:22.388  Established   
  BGP state:          Established
    Neighbor address: 169.254.100.254%ext-vlan
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      11.0.0.1
    Local capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Neighbor capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Session:          external AS4
    Source address:   169.254.100.3
    Hold timer:       2.270/3
    Keepalive timer:  0.556/1
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static
    Routes:         1 imported, 2 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              4          2          0        ---          2
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   169.254.100.3
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              3          0          3        ---          0
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   100:100::3 fe80::200:ff:fe01:102
        
NBR-gateway2 BGP        ---        up     16:03:25.264  Established   
  BGP state:          Established
    Neighbor address: fe80::beef%ext-vlan
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      11.0.0.1
    Local capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Neighbor capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Session:          external AS4
    Source address:   fe80::200:ff:fe01:102
    Hold timer:       2.242/3
    Keepalive timer:  0.635/1
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              3          0          3        ---          0
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   169.254.100.3
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static
    Routes:         1 imported, 1 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              3          1          1        ---          1
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   :: fe80::200:ff:fe01:102

NBR-gateway3 BGP        ---        start  17:17:33.913  Idle          Received: Hold timer expired
  BGP state:          Idle
    Neighbor address: 169.254.100.253%ext-vlan
    Neighbor AS:      4248829953
    Local AS:         8103
    Error wait:       43.061/60
    Last error:       Received: Hold timer expired
  Channel ipv4
    State:          DOWN
    Table:          master4
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static
  Channel ipv6
    State:          DOWN
    Table:          master6
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT
        
NBR-gateway4 BGP        ---        up     16:03:25.204  Established   
  BGP state:          Established
    Neighbor address: fe80::beee%ext-vlan
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      11.0.0.2
    Local capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Neighbor capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Session:          external AS4
    Source address:   fe80::200:ff:fe01:102
    Hold timer:       2.527/3
    Keepalive timer:  0.774/1
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              3          0          3        ---          0
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   169.254.100.3
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static
    Routes:         1 imported, 1 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              3          1          1        ---          1
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   :: fe80::200:ff:fe01:102

`

var bgpOutput string = `
BIRD 2.0.7 ready.
Name       Proto      Table      State  Since         Info
NBR-BFD    BFD        ---        up     16:03:17.278

NBR-gateway1 BGP        ---        up     16:03:22.388  Established
  BGP state:          Established
    Neighbor address: 169.254.100.254%ext-vlan
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      11.0.0.1
    Local capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Neighbor capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Session:          external AS4
    Source address:   169.254.100.3
    Hold timer:       1.915/3
    Keepalive timer:  0.117/1
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static
    Routes:         1 imported, 2 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              6          2          0        ---          4
      Export withdraws:            2        ---        ---        ---          2
    BGP Next hop:   169.254.100.3
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              6          0          6        ---          0
      Export withdraws:            2        ---        ---        ---          0
    BGP Next hop:   100:100::3 fe80::200:ff:fe01:102

NBR-gateway2 BGP        ---        up     17:53:39.604  Established
  BGP state:          Established
    Neighbor address: 100:100::254%ext-vlan
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      11.0.0.1
    Local capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Neighbor capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Session:          external AS4
    Source address:   100:100::3
    Hold timer:       2.224/3
    Keepalive timer:  0.775/1
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              3          0          3        ---          0
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   169.254.100.3
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static
    Routes:         1 imported, 1 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              3          2          0        ---          1
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   100:100::3 fe80::200:ff:fe01:102

NBR-gateway3 BGP        ---        up     17:18:30.468  Established
  BGP state:          Established
    Neighbor address: 169.254.100.253%ext-vlan
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      11.0.0.2
    Local capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Neighbor capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Session:          external AS4
    Source address:   169.254.100.3
    Hold timer:       2.120/3
    Keepalive timer:  0.005/1
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static
    Routes:         1 imported, 2 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              5          0          1        ---          4
      Export withdraws:            2        ---        ---        ---          2
    BGP Next hop:   169.254.100.3
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              5          0          5        ---          0
      Export withdraws:            2        ---        ---        ---          0
    BGP Next hop:   100:100::3 fe80::200:ff:fe01:102

NBR-gateway4 BGP        ---        up     17:53:40.211  Established
  BGP state:          Established
    Neighbor address: 100:100::253%ext-vlan
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      11.0.0.2
    Local capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Neighbor capabilities
      Multiprotocol
        AF announced: ipv4 ipv6
      Route refresh
      4-octet AS numbers
      Enhanced refresh
    Session:          external AS4
    Source address:   100:100::3
    Hold timer:       2.564/3
    Keepalive timer:  0.261/1
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              3          0          3        ---          0
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   169.254.100.3
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static
    Routes:         1 imported, 1 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              2          0          1        ---          1
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   100:100::3 fe80::200:ff:fe01:102

`
