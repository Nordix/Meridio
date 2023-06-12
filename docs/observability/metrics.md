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

### meridio.conduit.stream.status

Stream status in the conduit instance.

* Type: Gauge (Health Metric)
* Attributes:
   * Pod Name
   * Trench
   * Conduit
   * Stream

### meridio.conduit.stream.flow.matches

Counts number of packets that have matched a flow.

* Type: Counter
* Attributes:
   * Pod Name
   * Trench
   * Conduit
   * Stream
   * Flow

### meridio.conduit.stream.target.packet.hits

Counts number of packets that have hit a target.

* Type: Counter
* Attributes:
   * Pod Name
   * Trench
   * Conduit
   * Stream
   * Target (identifier + IPs)

### meridio.attractor.gateway.status

Gateway status in the attractor instance.

* Type: Gauge (Health Metric)
* Attributes:
   * Pod Name
   * Trench
   * Conduit
   * Attactor
   * Gateway
