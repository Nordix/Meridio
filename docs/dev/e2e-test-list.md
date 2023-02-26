# End-to-End Test List

## IngressTraffic 

### TCP-IPv4

```
IngressTraffic TCP-IPv4 when Send tcp traffic in trench-a with vip-1-v4 as destination IP and tcp-destination-port-0 as destination port (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/ingress_traffic_test.go:48
  STEP: Sending tcp traffic from the TG trench-a.red to 20.0.0.1:4000 @ 03/10/23 17:20:06.2827
```

### TCP-IPv6

```
IngressTraffic TCP-IPv6 when Send tcp traffic in trench-a with vip-1-v6 as destination IP and tcp-destination-port-0 as destination port (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/ingress_traffic_test.go:66
  STEP: Sending tcp traffic from the TG trench-a.red to [2000::1]:4000 @ 03/10/23 17:18:48.933
```

### UDP-IPv4

```
IngressTraffic UDP-IPv4 when Send udp traffic in trench-a with vip-1-v4 as destination IP and udp-destination-port-0 as destination port (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/ingress_traffic_test.go:84
  STEP: Sending udp traffic from the TG trench-a.red to 20.0.0.1:4003 @ 03/10/23 17:18:48.798
```

### UDP-IPv6

```
IngressTraffic UDP-IPv6 when Send udp traffic in trench-a with vip-1-v6 as destination IP and udp-destination-port-0 as destination port (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/ingress_traffic_test.go:102
  STEP: Sending udp traffic from the TG trench-a.red to [2000::1]:4003 @ 03/10/23 17:17:38.427
```

## MultiTrenches

### MT-Switch

```
MultiTrenches MT-Switch when Disconnect a target from target-a-deployment-name from trench-a and connect it to trench-b (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/multi_trenches_test.go:230
  STEP: Selecting the first target from the deployment with label app=target-a in namespace red @ 03/10/23 17:17:50.136
  STEP: Closing stream stream-a-i (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:17:50.143
  STEP: Opening stream stream-b-i (conduit: conduit-b-1, trench: trench-b) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:17:50.215
  STEP: Waiting the stream stream-a-i (conduit: conduit-a-1, trench: trench-a) to be closed and stream stream-b-i (conduit: conduit-b-1, trench: trench-b) to be opened in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:17:50.518
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 3 targets configured @ 03/10/23 17:17:53.678
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 3 targets configured @ 03/10/23 17:17:53.749
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-b-1-98db89789-pg76t (red) to have 5 targets configured @ 03/10/23 17:17:53.816
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-b-1-98db89789-rgj52 (red) to have 5 targets configured @ 03/10/23 17:17:53.885
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4000 @ 03/10/23 17:17:53.948
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4000 @ 03/10/23 17:17:54.107
  STEP: Sending tcp traffic from the TG trench-b (red) to 20.0.0.1:4000 @ 03/10/23 17:17:54.259
  STEP: Sending tcp traffic from the TG trench-b (red) to [2000::1]:4000 @ 03/10/23 17:17:54.434
  STEP: Closing stream stream-b-i (conduit: conduit-b-1, trench: trench-b) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:17:54.6
  STEP: Opening stream stream-a-i (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:17:54.669
  STEP: Waiting the stream stream-b-i (conduit: conduit-b-1, trench: trench-b) to be closed and stream stream-a-i (conduit: conduit-a-1, trench: trench-a) to be opened in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:17:54.978
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 4 targets configured @ 03/10/23 17:17:58.121
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 4 targets configured @ 03/10/23 17:17:58.18
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-b-1-98db89789-pg76t (red) to have 4 targets configured @ 03/10/23 17:17:58.258
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-b-1-98db89789-rgj52 (red) to have 4 targets configured @ 03/10/23 17:17:58.327
```

### MT-Parallel

```
MultiTrenches MT-Parallel when Send traffic in trench-a and trench-b at the same time (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/multi_trenches_test.go:86
  STEP: Sending tcp traffic from the TG trench-b (red) to 20.0.0.1:4000 @ 03/10/23 17:20:14.289
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4000 @ 03/10/23 17:20:14.289
  STEP: Sending tcp traffic from the TG trench-b (red) to [2000::1]:4000 @ 03/10/23 17:20:14.633
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4000 @ 03/10/23 17:20:14.633
```

## Scaling

### Scale-Down

```
Scaling Scale-Down when Scale down target-a-deployment-name (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/scaling_test.go:146
  STEP: Current list of targets: target-a-67b8f95485-4qjf5 target-a-67b8f95485-9px6t target-a-67b8f95485-c7tf9 target-a-67b8f95485-pms6r @ 03/10/23 17:17:58.406
  STEP: Scaling target-a deployment to 3 @ 03/10/23 17:17:58.406
  STEP: Waiting for the deployment target-a to be scaled to 3 @ 03/10/23 17:17:58.412
  STEP: Current list of targets: target-a-67b8f95485-4qjf5 target-a-67b8f95485-c7tf9 target-a-67b8f95485-pms6r @ 03/10/23 17:18:30.688
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 3 targets configured @ 03/10/23 17:18:30.693
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 3 targets configured @ 03/10/23 17:18:30.76
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4000 @ 03/10/23 17:18:30.828
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4000 @ 03/10/23 17:18:30.974
  STEP: Current list of targets: target-a-67b8f95485-4qjf5 target-a-67b8f95485-c7tf9 target-a-67b8f95485-pms6r @ 03/10/23 17:18:31.155
  STEP: Scaling target-a deployment to 4 @ 03/10/23 17:18:31.155
  STEP: Waiting for the deployment target-a to be scaled to 4 @ 03/10/23 17:18:31.159
  STEP: Current list of targets: target-a-67b8f95485-4qjf5 target-a-67b8f95485-5hwgq target-a-67b8f95485-c7tf9 target-a-67b8f95485-pms6r @ 03/10/23 17:18:43.316
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 4 targets configured @ 03/10/23 17:18:43.322
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 4 targets configured @ 03/10/23 17:18:43.386
```

### Scale-Up

```
Scaling Scale-Up when Scale up target-a-deployment-name (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/scaling_test.go:176
  STEP: Current list of targets: target-a-67b8f95485-4qjf5 target-a-67b8f95485-9px6t target-a-67b8f95485-c7tf9 target-a-67b8f95485-pms6r @ 03/10/23 17:16:42.659
  STEP: Scaling target-a deployment to 5 @ 03/10/23 17:16:42.659
  STEP: Waiting for the deployment target-a to be scaled to 5 @ 03/10/23 17:16:42.667
  STEP: Current list of targets: target-a-67b8f95485-4qjf5 target-a-67b8f95485-9px6t target-a-67b8f95485-c7tf9 target-a-67b8f95485-pms6r target-a-67b8f95485-wwnx5 @ 03/10/23 17:16:54.793
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 5 targets configured @ 03/10/23 17:16:54.798
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 5 targets configured @ 03/10/23 17:16:54.863
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4000 @ 03/10/23 17:16:54.938
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4000 @ 03/10/23 17:16:55.138
  STEP: Current list of targets: target-a-67b8f95485-4qjf5 target-a-67b8f95485-9px6t target-a-67b8f95485-c7tf9 target-a-67b8f95485-pms6r target-a-67b8f95485-wwnx5 @ 03/10/23 17:16:55.387
  STEP: Scaling target-a deployment to 4 @ 03/10/23 17:16:55.387
  STEP: Waiting for the deployment target-a to be scaled to 4 @ 03/10/23 17:16:55.392
  STEP: Current list of targets: target-a-67b8f95485-4qjf5 target-a-67b8f95485-9px6t target-a-67b8f95485-c7tf9 target-a-67b8f95485-pms6r @ 03/10/23 17:17:27.666
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 4 targets configured @ 03/10/23 17:17:27.672
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 4 targets configured @ 03/10/23 17:17:27.742
```

## TAPA

### close-open

```
TAPA close-open when Close stream-a-I in one of the target from target-a-deployment-name and re-open it (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/tapa_test.go:114
  STEP: Selecting the first target from the deployment with label app=target-a in namespace red @ 03/10/23 17:18:43.454
  STEP: Closing stream stream-a-i (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:18:43.46
  STEP: Waiting the stream to be closed in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:18:43.54
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 3 targets configured @ 03/10/23 17:18:44.146
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 3 targets configured @ 03/10/23 17:18:44.241
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4000 @ 03/10/23 17:18:44.746
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4000 @ 03/10/23 17:18:44.937
  STEP: Reopening stream stream-a-i (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:18:45.108
  STEP: Waiting the stream to be opened in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:18:45.483
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 4 targets configured @ 03/10/23 17:18:48.644
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 4 targets configured @ 03/10/23 17:18:48.722
```

## Trench

### delete-create-trench

```
Trench delete-create-trench when Delete trench-a and recreate and reconfigure it (Traffic) is received by the targets
/home/lionelj/Workspaces/Meridio/test/e2e/trench_test.go:56
  STEP: Deleting the trench trench-a @ 03/10/23 18:21:25.37
  STEP: Recreate the trench trench-a @ 03/10/23 18:21:59.724
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-f99d6d485-blj65 (red) to have 4 targets configured @ 03/10/23 18:22:42.78
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-f99d6d485-gvhkd (red) to have 4 targets configured @ 03/10/23 18:22:43.093
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4000 @ 03/10/23 18:22:43.174
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4000 @ 03/10/23 18:22:43.321
```

## Vip

### new-vip

```
Vip new-vip when Configure vip-2-v4 and vip-2-v6 in flow-a-z-tcp and attractor-a-1 (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/vip_test.go:44
  STEP: Configuring the new VIP @ 03/10/23 17:20:11.976
  STEP: Sending tcp traffic from the TG trench-a (red) to 60.0.0.150:4000 @ 03/10/23 17:20:12.248
  STEP: Sending tcp traffic from the TG trench-a (red) to [6000::150]:4000 @ 03/10/23 17:20:13.506
  STEP: Reverting the configuration of the new VIP @ 03/10/23 17:20:13.678
```

## Stream

### new-stream

```
Stream new-stream when Configure stream-a-III in conduit-a-1 with a new flow with tcp, tcp-destination-port-2 as destination port and vip-1-v4 and vip-1-v6 (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/stream_test.go:137
  STEP: Selecting the first target from the deployment with label app=target-a in namespace red @ 03/10/23 17:20:00.892
  STEP: Configuring the new stream @ 03/10/23 17:20:00.903
  STEP: Opening stream stream-a-iii (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:20:01.143
  STEP: Waiting the stream to be opened in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:20:01.253
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 1 targets configured @ 03/10/23 17:20:04.501
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 1 targets configured @ 03/10/23 17:20:04.569
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4002 @ 03/10/23 17:20:04.645
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4002 @ 03/10/23 17:20:04.866
  STEP: Reverting the configuration of the new stream @ 03/10/23 17:20:05.061
  STEP: Closing stream stream-a-iii (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:20:05.24
  STEP: Waiting the stream to be closed in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:20:05.345
  STEP: Waiting for nfqlb to have removed the stream configuration @ 03/10/23 17:20:05.974
  STEP: Waiting for nfqlb to have removed the stream configuration @ 03/10/23 17:20:06.126
```

### stream-max-targets

```

Stream stream-max-targets when Configure stream-a-III as in new-stream test with the max-targets field set to 1 and 2 targets with stream-a-III opened (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/stream_test.go:300
  STEP: Selecting the first target from the deployment with label app=target-a in namespace red @ 03/10/23 17:16:21.611
  STEP: Selecting the second target from the deployment with label app=target-a in namespace red @ 03/10/23 17:16:21.62
  STEP: Configuring the new stream with max-targets set to 1 @ 03/10/23 17:16:21.62
  STEP: Opening stream stream-a-iii (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:16:21.887
  STEP: Waiting the stream to be opened in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:16:22.107
  STEP: Opening stream stream-a-iii (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-9px6t in namespace red @ 03/10/23 17:16:25.386
  STEP: Waiting the stream to be unavailable in pod target-a-67b8f95485-9px6t using ./target-client watch @ 03/10/23 17:16:25.641
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-l9296 (red) to have 1 targets configured @ 03/10/23 17:16:29.023
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-1-7998554ffb-s6qb5 (red) to have 1 targets configured @ 03/10/23 17:16:29.157
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4002 @ 03/10/23 17:16:29.25
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4002 @ 03/10/23 17:16:29.532
  STEP: Reverting the configuration of the new stream @ 03/10/23 17:16:29.775
  STEP: Closing stream stream-a-iii (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-9px6t in namespace red @ 03/10/23 17:16:30.002
  STEP: Waiting the stream to be closed in pod target-a-67b8f95485-9px6t using ./target-client watch @ 03/10/23 17:16:30.151
  STEP: Closing stream stream-a-iii (conduit: conduit-a-1, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:16:30.773
  STEP: Waiting the stream to be closed in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:16:30.861
  STEP: Waiting for nfqlb to have removed the stream configuration @ 03/10/23 17:16:31.433
  STEP: Waiting for nfqlb to have removed the stream configuration @ 03/10/23 17:16:31.502
```

## Flow

### new-flow

```
Flow new-flow when Configure a new flow with tcp, tcp-destination-port-2 as destination port and vip-1-v4 and vip-1-v6 in stream-a-I (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/flow_test.go:44
  STEP: Configuring the new flow @ 03/10/23 17:17:27.806
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4002 @ 03/10/23 17:17:33.027
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4002 @ 03/10/23 17:17:33.197
  STEP: Reverting the configuration of the new flow @ 03/10/23 17:17:33.328
```

### flow-priority

```
Flow flow-priority when Set priority to 3 and add tcp-destination-port-1 as destination port in flow-a-z-tcp (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/flow_test.go:83
  STEP: Configuring the flow @ 03/10/23 17:19:49.958
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4001 @ 03/10/23 17:19:55.178
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4001 @ 03/10/23 17:19:55.345
  STEP: Reverting the configuration of the flow @ 03/10/23 17:19:55.535
```

### flow-destination-ports-range

```
Flow flow-destination-ports-range when Set priority to 3 and set 'tcp-destination-port-0'-'tcp-destination-port-2' as destination port in flow-a-z-tcp (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/flow_test.go:122
  STEP: Configuring the flow @ 03/10/23 17:17:38.553
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4000 @ 03/10/23 17:17:43.787
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4000 @ 03/10/23 17:17:43.961
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4001 @ 03/10/23 17:17:44.144
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4001 @ 03/10/23 17:17:44.295
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4002 @ 03/10/23 17:17:44.471
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4002 @ 03/10/23 17:17:44.617
  STEP: Reverting the configuration of the flow @ 03/10/23 17:17:44.814
```

### flow-byte-matches

```
Flow flow-byte-matches when Add tcp-destination-port-2 to destination ports of flow-a-z-tcp and add a byte-match to allow only tcp-destination-port-2 (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/flow_test.go:163
  STEP: Configuring the flow @ 03/10/23 17:20:15.06
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4000 @ 03/10/23 17:20:20.335
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4000 @ 03/10/23 17:20:21.433
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:4002 @ 03/10/23 17:20:22.558
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:4002 @ 03/10/23 17:20:22.732
  STEP: Reverting the configuration of the flow @ 03/10/23 17:20:22.886
```

## Attractor

### new-attractor-nsm-vlan

```
Attractor new-attractor-nsm-vlan when Configure a new attractor with new vips vip-2-v4 and vip-2-v6, gateways, conduit conduit-a-3, stream stream-a-III and flow with tcp and tcp-destination-port-0 as destination port (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/attractor_test.go:119
  STEP: Selecting the first target from the deployment with label app=target-a in namespace red @ 03/10/23 17:18:53.995
  STEP: Configuring the new attractor @ 03/10/23 17:18:54.015
  STEP: Opening stream stream-a-iii (conduit: conduit-a-3, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:19:17.055
  STEP: Waiting the stream to be opened in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:19:17.191
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-3-59fd97fb4-nrd4h (red) to have 1 targets configured @ 03/10/23 17:19:20.507
  STEP: Waiting for nfqlb in the stateless-lb-frontend-attractor-a-3-59fd97fb4-sbbn5 (red) to have 1 targets configured @ 03/10/23 17:19:20.629
  STEP: Sending tcp traffic from the TG trench-a (red) to 60.0.0.150:4000 @ 03/10/23 17:19:20.739
  STEP: Sending tcp traffic from the TG trench-a (red) to [6000::150]:4000 @ 03/10/23 17:19:20.985
  STEP: Closing stream stream-a-iii (conduit: conduit-a-3, trench: trench-a) in target target-a-67b8f95485-4qjf5 in namespace red @ 03/10/23 17:19:21.261
  STEP: Waiting the stream to be closed in pod target-a-67b8f95485-4qjf5 using ./target-client watch @ 03/10/23 17:19:21.371
  STEP: Reverting the configuration of the new attractor @ 03/10/23 17:19:22.021
```

## Conduit

### conduit-destination-port-nats

```
Conduit conduit-destination-port-nats when Configure flow-a-z-tcp with tcp-destination-port-nat-0 as destination port and conduit-a-1 with a port nat with tcp-destination-port-nat-0 as port and tcp-destination-port-0 as target-port (Traffic) is received by the targets
/home/jenkins/nordix/slave_root/workspace/meridio-e2e-test-kind/11891/test/e2e/conduit_test.go:44
  STEP: Configuring the Conduit @ 03/10/23 17:16:31.572
  STEP: Sending tcp traffic from the TG trench-a (red) to 20.0.0.1:80 @ 03/10/23 17:16:36.824
  STEP: Sending tcp traffic from the TG trench-a (red) to [2000::1]:80 @ 03/10/23 17:16:37.063
  STEP: Reverting the configuration of the Conduit @ 03/10/23 17:16:37.291
```
