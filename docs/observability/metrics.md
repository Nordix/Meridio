# Metrics

## Metric List

### meridio.interface.`METRIC_TYPE`

`METRIC_TYPE`: rx_packets, tx_packets, rx_bytes, tx_bytes, rx_errors, tx_errors, rx_dropped, tx_dropped

Counts number of `METRIC_TYPE` for a network interface.

* Type: Counter
* Attributes:
   * Pod Name
   * Trench
   * Conduit (optional)
   * Attactor (optional)
   * Interface Name

### meridio.conduit.stream.flow.matches

Counts number of packets that have matched a flow.

* Type: Counter
* Attributes:
   * Pod Name
   * Trench
   * Conduit
   * Stream
   * Flow

### meridio.conduit.stream.target.hits.packets

Counts number of packets that have hit a target.

* Type: Counter
* Attributes:
   * Pod Name
   * Trench
   * Conduit
   * Stream
   * Target (identifier + IPs)

### meridio.conduit.stream.target.hits.bytes

Counts number of bytes that have hit a target.

* Type: Counter
* Attributes:
   * Pod Name
   * Trench
   * Conduit
   * Stream
   * Target (identifier + IPs)

### meridio.conduit.stream.target.latency (Planned)

Reports the latency with a target.

* Type: Gauge
* Attributes:
   * Pod Name
   * Trench
   * Conduit
   * IP

### meridio.attractor.gateway.status (Planned)

Gateway status in the attractor instance.

* Type: Gauge (Health Metric)
* Attributes:
   * Pod Name
   * Trench
   * Conduit
   * Attactor
   * Gateway
