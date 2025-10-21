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

var extInterface string = `eth0.100`

type testGateway struct {
	ip  string
	af  int
	bfd bool
}

var configuredGatewayNamesByFamily map[string]*testGateway = map[string]*testGateway{}

func getGatewayByName(name string) (*testGateway, bool) {
	gw, ok := configuredGatewayNamesByFamily[name]
	return gw, ok
}

// parse protocol outputs to determine connectivity
func check(protocolOutput, bfdOutput string, cs *connectivity.ConnectivityStatus, logp *string) {
	bird.ParseProtocols(protocolOutput, logp, func(name string, options ...bird.Option) {
		gw, ok := getGatewayByName(name)
		if !ok {
			// no configured gateway found for the name
			return
		}
		ip := gw.ip
		family := gw.af
		bfd := gw.bfd

		// extend protocol options with external interface, gateway ip, bfd sessions
		opts := append([]bird.Option{
			bird.WithInterface(extInterface),
			bird.WithNeighbor(ip),
			bird.WithBfdSessions(bfdOutput),
			bird.WithBfd(bfd),
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
// 1 IPv4 and 1 IPv6 BGP protocol sessions are present in the output. All the sessions are up.
// All sessions are known to the configuration. BFD is not configured.
//
// After parsing:
// External connectivity must be OK.
// All BGP sessions known to configuration must be up.
// Logs collected by the parser must contain all BGP sessions.
func TestParseProtocols(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	configuredGatewayNamesByFamily = map[string]*testGateway{
		`NBR-gateway1`: {ip: `169.254.100.150`, af: syscall.AF_INET, bfd: true},
		`NBR-gateway2`: {ip: `100:100::150`, af: syscall.AF_INET6, bfd: true},
	}
	expectedLog := `BIRD 3.1.4 ready.` + "\n" +
		`Name       Proto      Table      State  Since         Info` + "\n" +
		`NBR-gateway1 BGP        ---        up     12:17:33.417  Established       Neighbor address: 169.254.100.150%eth0.100` + "\n" +
		`NBR-gateway2 BGP        ---        up     12:17:34.062  Established       Neighbor address: 100:100::150%eth0.100` + "\n"

	assert.NotNil(t, cs)
	assert.Equal(t, "", cs.Log())

	check(bgpOutput, "", cs, cs.Logp())

	t.Logf("cs: %v\n", cs)
	assert.False(t, cs.NoConnectivity())
	assert.False(t, cs.AnyGatewayDown())
	assert.Equal(t, expectedLog, cs.Log())
}

// TestParseProtocolsWithBfd -
// 1 IPv4 and 1 IPv6 (link-local addr) BGP protocol sessions are present in the output.
// All sessions are known to the configuration (configuredGatewayNamesByFamily).
// BFD is configured for all the neighbors, and all are up.
//
// After parsing:
// External connectivity must be OK.
// All BGP sessions part of the configuration must be up.
func TestParseProtocolsWithBfd(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	cs := connectivity.NewConnectivityStatus()
	configuredGatewayNamesByFamily = map[string]*testGateway{
		`NBR-gateway1`: {ip: `169.254.100.150`, af: syscall.AF_INET, bfd: true},
		`NBR-gateway2`: {ip: `fe80::98c6:29ff:fe52:ccb5`, af: syscall.AF_INET6, bfd: true},
	}
	bfdOutput := `
		BIRD 3.1.4 ready.
    NBR-BFD:
    IP address                Interface  State      Since         Interval  Timeout
    169.254.100.150           eth0.100   Up         14:34:31.904    0.300    1.500
    fe80::98c6:29ff:fe52:ccb5 eth0.100   Up         14:34:27.614    0.300    1.500
	`

	assert.NotNil(t, cs)
	assert.Empty(t, cs.Log())

	check(bgpLinkLocalOutput, bfdOutput, cs, nil)

	t.Logf("cs: %v\n", cs)
	assert.False(t, cs.NoConnectivity())
	assert.False(t, cs.AnyGatewayDown())
}

var bgpLinkLocalOutput string = `
BIRD 3.1.4 ready.
Name       Proto      Table      State  Since         Info
NBR-gateway1 BGP        ---        up     14:34:31.904  Established   
  Created:            14:34:27.614
  BGP state:          Established
    Neighbor address: 169.254.100.150%eth0.100
    Neighbor port:    10179
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      169.254.100.150
    Local capabilities
      Multiprotocol
        AF announced: ipv4
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
    Source address:   169.254.100.2
    Hold timer:       2.089/3
    Keepalive timer:  0.448/1
    TX pending:       0 bytes
    Send hold timer:  4.863/6
  Channel ipv4
    State:          UP
    Import state:   UP
    Export state:   READY
    Table:          master4
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   RX limit      limit   accepted
      Import updates:              1          0          0          0          0          0          1
      Import withdraws:            0          0        ---          0        ---        ---          0
      Export updates:              1          1          0          0        ---          0          0
      Export withdraws:            0        ---        ---          0        ---        ---          0
    BGP Next hop:   169.254.100.2
    Pending 0 attribute sets with total 0 prefixes to send

NBR-gateway2 BGP        ---        up  14:34:27.614  Established
  Created:            14:34:27.614
  BGP state:          Established
    Neighbor address: fe80::98c6:29ff:fe52:ccb5%eth0.100
    Neighbor AS:      4248829953
    Local AS:         8103
    Connect delay:    3.942/5
    Last error:       Socket: Connection reset by peer
      Local capabilities
      Multiprotocol
        AF announced: ipv6
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
    Source address:   100:100::2
    Hold timer:       1.736/3
    Keepalive timer:  0.394/1
    TX pending:       0 bytes
    Send hold timer:  4.870/6
  Channel ipv6
    State:          UP
    Import state:   UP
    Export state:   READY
    Table:          master6
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access
    Routes:         1 imported, 1 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   RX limit      limit   accepted
      Import updates:              1          0          0          0          0          0          1
      Import withdraws:            0          0        ---          0        ---        ---          0
      Export updates:              2          1          0          0        ---          0          1
      Export withdraws:            0        ---        ---          0        ---        ---          0
    BGP Next hop:   100:100::2 fe80::70bd:69ff:fe9f:8f00
    Pending 0 attribute sets with total 0 prefixes to send
    
NBR-BFD    BFD        ---        up     14:34:17.713  
  Created:            14:34:17.713
`

var bgpOutput string = `
BIRD 3.1.4 ready.
Name       Proto      Table      State  Since         Info
NBR-gateway1 BGP        ---        up     12:17:33.417  Established   
  Created:            12:17:29.169
  BGP state:          Established
    Neighbor address: 169.254.100.150%eth0.100
    Neighbor port:    10179
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      169.254.100.150
    Local capabilities
      Multiprotocol
        AF announced: ipv4
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
    Source address:   169.254.100.2
    Hold timer:       2.715/3
    Keepalive timer:  0.391/1
    TX pending:       0 bytes
    Send hold timer:  4.428/6
  Channel ipv4
    State:          UP
    Import state:   UP
    Export state:   READY
    Table:          master4
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access
    Routes:         1 imported, 2 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   RX limit      limit   accepted
      Import updates:              1          0          0          0          0          0          1
      Import withdraws:            0          0        ---          0        ---        ---          0
      Export updates:              3          1          0          0        ---          0          2
      Export withdraws:            0        ---        ---          0        ---        ---          0
    BGP Next hop:   169.254.100.2
    Pending 0 attribute sets with total 0 prefixes to send

NBR-gateway2 BGP        ---        up     12:17:34.062  Established   
  Created:            12:17:29.169
  BGP state:          Established
    Neighbor address: 100:100::150%eth0.100
    Neighbor port:    10179
    Neighbor AS:      4248829953
    Local AS:         8103
    Neighbor ID:      169.254.100.150
    Local capabilities
      Multiprotocol
        AF announced: ipv6
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
    Source address:   100:100::2
    Hold timer:       1.736/3
    Keepalive timer:  0.394/1
    TX pending:       0 bytes
    Send hold timer:  4.870/6
  Channel ipv6
    State:          UP
    Import state:   UP
    Export state:   READY
    Table:          master6
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access
    Routes:         1 imported, 1 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   RX limit      limit   accepted
      Import updates:              1          0          0          0          0          0          1
      Import withdraws:            0          0        ---          0        ---        ---          0
      Export updates:              2          1          0          0        ---          0          1
      Export withdraws:            0        ---        ---          0        ---        ---          0
    BGP Next hop:   100:100::2 fe80::70bd:69ff:fe9f:8f00
    Pending 0 attribute sets with total 0 prefixes to send

NBR-BFD    BFD        ---        up     12:17:19.296  
  Created:            12:17:19.296
`
