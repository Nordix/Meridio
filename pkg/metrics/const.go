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

package metrics

const (
	MERIDIO_CONDUIT_STREAM_FLOW_MATCHES        = "meridio.conduit.stream.flow.matches"
	MERIDIO_CONDUIT_STREAM_TARGET_HIT_PACKETS  = "meridio.conduit.stream.target.hit.packets"
	MERIDIO_CONDUIT_STREAM_TARGET_HIT_BYTES    = "meridio.conduit.stream.target.hit.bytes"
	MERIDIO_INTERFACE_RX_PACKETS               = "meridio.interface.rx_packets"
	MERIDIO_INTERFACE_TX_PACKET                = "meridio.interface.tx_packets"
	MERIDIO_INTERFACE_RX_BYTES                 = "meridio.interface.rx_bytes"
	MERIDIO_INTERFACE_TX_BYTES                 = "meridio.interface.tx_bytes"
	MERIDIO_INTERFACE_RX_ERRORS                = "meridio.interface.rx_errors"
	MERIDIO_INTERFACE_TX_ERRORS                = "meridio.interface.tx_errors"
	MERIDIO_INTERFACE_RX_DROPPED               = "meridio.interface.rx_dropped"
	MERIDIO_INTERFACE_TX_DROPPED               = "meridio.interface.tx_dropped"
	MERIDIO_ATTRACTOR_GATEWAY_IMPORTED_ROUTES  = "meridio.attractor.gateway.imported.routes"
	MERIDIO_ATTRACTOR_GATEWAY_EXPORTED_ROUTES  = "meridio.attractor.gateway.exported.routes"
	MERIDIO_ATTRACTOR_GATEWAY_PREFERRED_ROUTES = "meridio.attractor.gateway.preferred.routes"

	METER_NAME = "Meridio"
)
