# Solution

```
cat <<EOF | kubectl apply -f -
---
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  name: gateway-v4-a
  namespace: red
  labels:
    trench: trench-a
spec:
  address: 169.254.100.150
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: 24s
    local-port: 10179
    remote-port: 10179
    bfd:
      switch: true
      min-tx: 300ms
      min-rx: 300ms
      multiplier: 5
---
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  name: gateway-v6-a
  namespace: red
  labels:
    trench: trench-a
spec:
  address: 100:100::150
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: 24s
    local-port: 10179
    remote-port: 10179
    bfd:
      switch: true
      min-tx: 300ms
      min-rx: 300ms
      multiplier: 5
EOF
```{{exec}}

# Explanation

The `local-port` and `remote-port` used by the gateways were configured to use `10180` instead of `10179`.