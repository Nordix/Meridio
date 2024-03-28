# Metrics

## Metric List

### meridio.interface.`METRIC_TYPE`

`METRIC_TYPE`: rx_packets, tx_packets, rx_bytes, tx_bytes, rx_errors, tx_errors, rx_dropped, tx_dropped

Counts number of `METRIC_TYPE` for a network interface.

* Type: Counter
* Attributes:
   * Trench
   * Conduit (optional)
   * Attactor (optional)
   * Interface Name
   * MAC Address
   * IPs

### meridio.conduit.stream.flow.matches

Counts number of packets that have matched a flow.

* Type: Counter
* Attributes:
   * Trench
   * Conduit
   * Stream
   * Flow

### meridio.conduit.stream.target.hit.`METRIC_TYPE`

`METRIC_TYPE`: packets, bytes

Counts number of `METRIC_TYPE` that have hit a target.

* Type: Counter
* Attributes:
   * Trench
   * Conduit
   * Stream
   * Identifier
   * Target IPs

### meridio.conduit.stream.target.latency (Planned)

Reports the latency with a target.

* Type: Gauge
* Attributes:
   * Trench
   * Conduit
   * Target IP

### meridio.attractor.gateway.`METRIC_TYPE`.routes

`METRIC_TYPE`: imported, exported, preferred

Number of `METRIC_TYPE` routes for a gateway in the attractor instance.

* Type: Gauge
* Attributes:
   * Trench
   * Attactor
   * Gateway
   * Gateway IP
   * Protocol
