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

package frontend

import (
	"reflect"
	"testing"
)

func TestParseShowProtocolsAll(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name string
		args args
		want []*BirdStats
	}{
		{
			name: "t1",
			args: args{
				output: t1,
			},
			want: []*BirdStats{
				{
					gatewayName:     "device1",
					routesImported:  0,
					routesExported:  0,
					routesPreferred: 0,
				},
				{
					gatewayName:     "VIP4",
					routesImported:  1,
					routesExported:  0,
					routesPreferred: 1,
				},
				{
					gatewayName:     "VIP6",
					routesImported:  1,
					routesExported:  0,
					routesPreferred: 1,
				},
				{
					gatewayName:     "kernel1",
					routesImported:  0,
					routesExported:  1,
					routesPreferred: 0,
				},
				{
					gatewayName:     "kernel2",
					routesImported:  0,
					routesExported:  1,
					routesPreferred: 0,
				},
				{
					gatewayName:     "DROP4",
					routesImported:  1,
					routesExported:  0,
					routesPreferred: 1,
				},
				{
					gatewayName:     "DROP6",
					routesImported:  1,
					routesExported:  0,
					routesPreferred: 1,
				},
				{
					gatewayName:     "kernel3",
					routesImported:  0,
					routesExported:  1,
					routesPreferred: 0,
				},
				{
					gatewayName:     "kernel4",
					routesImported:  0,
					routesExported:  1,
					routesPreferred: 0,
				},
				{
					gatewayName:     "NBR-gateway-v4-a-1",
					routesImported:  1,
					routesExported:  1,
					routesPreferred: 1,
				},
				{
					gatewayName:     "NBR-gateway-v6-a-1",
					routesImported:  1,
					routesExported:  1,
					routesPreferred: 1,
				},
				{
					gatewayName:     "NBR-BFD",
					routesImported:  0,
					routesExported:  0,
					routesPreferred: 0,
				},
			},
		},
		{
			name: "t2",
			args: args{
				output: t2,
			},
			want: []*BirdStats{
				{
					gatewayName:     "device1",
					routesImported:  0,
					routesExported:  0,
					routesPreferred: 0,
				},
				{
					gatewayName:     "kernel1",
					routesImported:  0,
					routesExported:  1,
					routesPreferred: 0,
				},
				{
					gatewayName:     "kernel2",
					routesImported:  0,
					routesExported:  1,
					routesPreferred: 0,
				},
				{
					gatewayName:     "DROP4",
					routesImported:  1,
					routesExported:  0,
					routesPreferred: 1,
				},
				{
					gatewayName:     "DROP6",
					routesImported:  1,
					routesExported:  0,
					routesPreferred: 1,
				},
				{
					gatewayName:     "kernel3",
					routesImported:  0,
					routesExported:  0,
					routesPreferred: 0,
				},
				{
					gatewayName:     "kernel4",
					routesImported:  0,
					routesExported:  0,
					routesPreferred: 0,
				},
				{
					gatewayName:     "NBR-gateway-v4-a-1",
					routesImported:  0,
					routesExported:  0,
					routesPreferred: 0,
				},
				{
					gatewayName:     "NBR-gateway-v6-a-1",
					routesImported:  0,
					routesExported:  0,
					routesPreferred: 0,
				},
				{
					gatewayName:     "NBR-BFD",
					routesImported:  0,
					routesExported:  0,
					routesPreferred: 0,
				},
			},
		},
		{
			name: "t3",
			args: args{
				output: t3,
			},
			want: []*BirdStats{
				{
					gatewayName:     "NBR-gateway-v4-a-1",
					routesImported:  1,
					routesExported:  1,
					routesPreferred: 1,
				},
				{
					gatewayName:     "NBR-gateway-v6-a-1",
					routesImported:  1,
					routesExported:  1,
					routesPreferred: 1,
				},
				{
					gatewayName:     "NBR-BFD",
					routesImported:  0,
					routesExported:  0,
					routesPreferred: 0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseShowProtocolsAll(tt.args.output); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseShowProtocolsAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

var t1 = `BIRD 2.0.10 ready.
Name       Proto      Table      State  Since         Info
device1    Device     ---        up     09:19:34.280  

VIP4       Static     master4    up     09:40:08.628  
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     110
    Input filter:   ACCEPT
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

VIP6       Static     master6    up     09:40:08.628  
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     110
    Input filter:   ACCEPT
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

kernel1    Kernel     drop4      up     09:19:43.404  
  Channel ipv4
    State:          UP
    Table:          drop4
    Preference:     10
    Input filter:   REJECT
    Output filter:  ACCEPT
    Routes:         0 imported, 1 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              1          0          0        ---          1
      Export withdraws:            0        ---        ---        ---          0

kernel2    Kernel     drop6      up     09:19:43.404  
  Channel ipv6
    State:          UP
    Table:          drop6
    Preference:     10
    Input filter:   REJECT
    Output filter:  ACCEPT
    Routes:         0 imported, 1 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              1          0          0        ---          1
      Export withdraws:            0        ---        ---        ---          0

DROP4      Static     drop4      up     09:19:43.403  
  Channel ipv4
    State:          UP
    Table:          drop4
    Preference:     0
    Input filter:   ACCEPT
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

DROP6      Static     drop6      up     09:19:43.403  
  Channel ipv6
    State:          UP
    Table:          drop6
    Preference:     0
    Input filter:   ACCEPT
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

kernel3    Kernel     master4    up     09:19:43.403  
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     10
    Input filter:   REJECT
    Output filter:  cluster_breakout
    Routes:         0 imported, 1 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              6          0          3        ---          3
      Export withdraws:            4        ---        ---        ---          2

kernel4    Kernel     master6    up     09:19:43.403  
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     10
    Input filter:   REJECT
    Output filter:  cluster_breakout
    Routes:         0 imported, 1 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              6          0          3        ---          3
      Export withdraws:            4        ---        ---        ---          2

NBR-gateway-v4-a-1 BGP        ---        up     09:40:08.169  Established   
  BGP state:          Established
    Neighbor address: 169.254.100.150%ext-vlan0
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
    Source address:   169.254.100.1
    Hold timer:       1.724/3
    Keepalive timer:  0.746/1
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access
    Routes:         1 imported, 1 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              2          1          0        ---          1
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   169.254.100.1

NBR-gateway-v6-a-1 BGP        ---        up     09:40:08.002  Established   
  BGP state:          Established
    Neighbor address: 100:100::150%ext-vlan0
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
    Source address:   100:100::1
    Hold timer:       2.506/3
    Keepalive timer:  0.233/1
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access
    Routes:         1 imported, 1 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              2          1          0        ---          1
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   100:100::1 fe80::fe:34ff:fe07:4bc6

NBR-BFD    BFD        ---        up     09:19:34.280`

var t2 = `BIRD 2.0.10 ready.
Name       Proto      Table      State  Since         Info
device1    Device     ---        up     09:19:34.280  

kernel1    Kernel     drop4      up     09:19:43.404  
  Channel ipv4
    State:          UP
    Table:          drop4
    Preference:     10
    Input filter:   REJECT
    Output filter:  ACCEPT
    Routes:         0 imported, 1 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              1          0          0        ---          1
      Export withdraws:            0        ---        ---        ---          0

kernel2    Kernel     drop6      up     09:19:43.404  
  Channel ipv6
    State:          UP
    Table:          drop6
    Preference:     10
    Input filter:   REJECT
    Output filter:  ACCEPT
    Routes:         0 imported, 1 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              1          0          0        ---          1
      Export withdraws:            0        ---        ---        ---          0

DROP4      Static     drop4      up     09:19:43.403  
  Channel ipv4
    State:          UP
    Table:          drop4
    Preference:     0
    Input filter:   ACCEPT
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

DROP6      Static     drop6      up     09:19:43.403  
  Channel ipv6
    State:          UP
    Table:          drop6
    Preference:     0
    Input filter:   ACCEPT
    Output filter:  REJECT
    Routes:         1 imported, 0 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

kernel3    Kernel     master4    up     09:19:43.403  
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     10
    Input filter:   REJECT
    Output filter:  cluster_breakout
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              6          0          3        ---          3
      Export withdraws:            6        ---        ---        ---          3

kernel4    Kernel     master6    up     09:19:43.403  
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     10
    Input filter:   REJECT
    Output filter:  cluster_breakout
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              6          0          3        ---          3
      Export withdraws:            6        ---        ---        ---          3

NBR-gateway-v4-a-1 BGP        ---        start  10:39:56.122  Active        Socket: Connection closed
  BGP state:          Active
    Neighbor address: 169.254.100.150%ext-vlan0
    Neighbor AS:      4248829953
    Local AS:         8103
    Connect delay:    3.509/5
    Last error:       Socket: Connection closed
  Channel ipv4
    State:          DOWN
    Table:          master4
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access

NBR-gateway-v6-a-1 BGP        ---        start  10:39:56.122  Active        Socket: Connection closed
  BGP state:          Active
    Neighbor address: 100:100::150%ext-vlan0
    Neighbor AS:      4248829953
    Local AS:         8103
    Connect delay:    3.505/5
    Last error:       Socket: Connection closed
  Channel ipv6
    State:          DOWN
    Table:          master6
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access

NBR-BFD    BFD        ---        up     09:19:34.280`

var t3 = `BIRD 2.0.10 ready.
Name       Proto      Table      State  Since         Info
NBR-gateway-v4-a-1 BGP        ---        up     13:49:37.427  Established   
  BGP state:          Established
    Neighbor address: 169.254.100.150%ext-vlan0
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
    Source address:   169.254.100.1
    Hold timer:       2.107/3
    Keepalive timer:  0.268/1
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access
    Routes:         1 imported, 1 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              2          1          0        ---          1
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   169.254.100.1

NBR-gateway-v6-a-1 BGP        ---        up     13:49:37.047  Established   
  BGP state:          Established
    Neighbor address: 100:100::150%ext-vlan0
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
    Source address:   100:100::1
    Hold timer:       2.193/3
    Keepalive timer:  0.055/1
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     100
    Input filter:   cluster_breakout
    Output filter:  cluster_access
    Routes:         1 imported, 1 exported, 1 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              1          0          0          0          1
      Import withdraws:            0          0        ---          0          0
      Export updates:              2          1          0        ---          1
      Export withdraws:            0        ---        ---        ---          0
    BGP Next hop:   100:100::1 fe80::fe:a6ff:fea1:960a

NBR-BFD    BFD        ---        up     13:49:34.392  `
