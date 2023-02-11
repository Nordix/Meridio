# Solution

```
cat <<EOF | kubectl apply -f -
---
apiVersion: meridio.nordix.org/v1
kind: Flow
metadata:
  name: flow-a-z-tcp
  namespace: red
  labels:
    trench: trench-a
spec:
  stream: stream-a-i
  priority: 1
  vips:
  - vip-a-1-v4
  - vip-a-1-v6
  source-subnets:
  - 0.0.0.0/0
  - 0:0:0:0:0:0:0:0/0
  source-ports:
  - any
  destination-ports:
  - "4000"
  protocols:
  - tcp
EOF
```{{exec}}

# Explanation

All custom resources were configured correctly except `flow-a-z-tcp` which was steering the destination port `400` instead of `4000`.
