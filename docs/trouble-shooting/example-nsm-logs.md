# Main Event Logging in NSM

This section presents how to check the logs for some major events in NSM.

Note that the following examples reflect the state of NSM 1.4.0 and can be changed.

## Startup

Command to filter out logs related to completed startup of each NSM element;

```bash
grep "tartup completed" *
```

Example output;

```bash
forwarder-vpp-jk2bt.log:Jul 21 12:44:57.092 [INFO] [cmd:/bin/forwarder] Startup completed in 8.987392153s
nse-remote-vlan-66bfccf745-qmx22.log:Jul 21 12:44:56.788 [INFO] [cmd:/bin/app] startup completed in 11.2814327s
nsmgr-rpb7r-nsmgr.log:Jul 21 12:44:56.437 [INFO] (1.3)   Startup completed in 11.051687112s
registry-k8s-7974cd559f-mbcw6.log:Jul 21 12:44:56.573 [INFO] [cmd:/bin/cmd-registry-k8s] Startup completed in 11.043761452s
```

Command to check the current config in NSM logs:

```bash
egrep "Config:|configuration" *
```

Example output;

```bash
forwarder-vpp-jk2bt.log:Jul 21 12:44:48.106 [INFO] [cmd:/bin/forwarder] Config: &config.Config{Name:"forwarder-vpp-jk2bt", Labels:map[string]string{"p2p":"true"}, NSName:"forwarder", ConnectTo:url.URL{Scheme:"unix", Opaque:"", User:(*url.Userinfo)(nil), Host:"", Path:"/var/lib/networkservicemesh/nsm.io.sock", RawPath:"", ForceQuery:false, RawQuery:"", Fragment:"", RawFragment:""}, ListenOn:url.URL{Scheme:"unix", Opaque:"", User:(*url.Userinfo)(nil), Host:"", Path:"/listen.on.sock", RawPath:"", ForceQuery:false, RawQuery:"", Fragment:"", RawFragment:""}, MaxTokenLifetime:600000000000, LogLevel:"TRACE", DialTimeout:100000000, OpenTelemetryEndpoint:"otel-collector.observability.svc.cluster.local:4317", TunnelIP:net.IP{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xac, 0x12, 0x0, 0x2}, VxlanPort:0x0, VppAPISocket:"/var/run/vpp/external/vpp-api.sock", VppInit:vppinit.Func{f:(func(context.Context, api.Connection, net.IP) (net.IP, error))(0xc86720)}, ResourcePollTimeout:30000000000, DevicePluginPath:"/var/lib/kubelet/device-plugins/", PodResourcesPath:"/var/lib/kubelet/pod-resources/", DeviceSelectorFile:"/var/lib/networkservicemesh/device-selector.yaml", SRIOVConfigFile:"", PCIDevicesPath:"/sys/bus/pci/devices", PCIDriversPath:"/sys/bus/pci/drivers", CgroupPath:"/host/sys/fs/cgroup/devices", VFIOPath:"/host/dev/vfio"}
forwarder-vpp-jk2bt.log:Jul 21 12:44:48.106 [INFO] [Config:ReadConfig] unmarshalled Config: &{Interfaces:[&{Name:ext_net1 Matches:[&{LabelSelector[&{Via:gw1}]}]}]}
nse-remote-vlan-66bfccf745-qmx22.log:Jul 21 12:44:45.507 [INFO] [cmd:/bin/app] Config: &config.Config{Name:"nse-remote-vlan-66bfccf745-qmx22", ConnectTo:url.URL{Scheme:"registry", Opaque:"5002", User:(*url.Userinfo)(nil), Host:"", Path:"", RawPath:"", ForceQuery:false, RawQuery:"", Fragment:"", RawFragment:""}, MaxTokenLifetime:60000000000, CidrPrefix:[]string{"172.10.0.0/24", "100:200::/64"}, RegisterService:true, ListenOn:url.URL{Scheme:"tcp", Opaque:"", User:(*url.Userinfo)(nil), Host:":5003", Path:"", RawPath:"", ForceQuery:false, RawQuery:"", Fragment:"", RawFragment:""}, OpenTelemetryEndpoint:"otel-collector.observability.svc.cluster.local:4317", LogLevel:"TRACE", Services:[]config.ServiceConfig{config.ServiceConfig{Name:"finance-bridge", Domain:"", Via:"gw1", VLANTag:100, Labels:map[string]string(nil)}}}
nsmgr-rpb7r-nsmgr.log:Jul 21 12:44:45.385 [INFO] (1.2)   Using configuration: &{nsmgr-rpb7r [{unix    /var/lib/networkservicemesh/nsm.io.sock  false   } {tcp   :5001   false   }] {registry 5002     false   } 10m0s TRACE 100ms forwarder otel-collector.observability.svc.cluster.local:4317}
registry-k8s-7974cd559f-mbcw6.log:Jul 21 12:44:45.530 [INFO] [cmd:/bin/cmd-registry-k8s] Config: &main.Config{Config:registryk8s.Config{Namespace:"nsm-system", ProxyRegistryURL:(*url.URL)(0xc000550090), ExpirePeriod:60000000000, ChainCtx:context.Context(nil), ClientSet:versioned.Interface(nil)}, ListenOn:[]url.URL{url.URL{Scheme:"tcp", Opaque:"", User:(*url.Userinfo)(nil), Host:":5002", Path:"", RawPath:"", ForceQuery:false, RawQuery:"", Fragment:"", RawFragment:""}}, LogLevel:"TRACE", OpenTelemetryEndpoint:"otel-collector.observability.svc.cluster.local:4317"}
```

## Service Registration

 Service registration logs are on 'TRACE' level.

 Run the following command to inspect the registry logs for;

- request

    ```bash
     grep "(1.1)   register={\"name\"" registry-k8s-7974cd559f-mbcw6.log
    ```

    Example output:

    ```bash
    Jul 21 12:44:56.769 [TRAC] [type:registry] (1.1)   register={"name":"finance-bridge","payload":"ETHERNET"}
    Jul 21 12:44:56.778 [TRAC] [type:registry] (1.1)   register={"name":"nse-remote-vlan-66bfccf745-qmx22","network_service_names":["finance-bridge"],"network_service_labels":{"finance-bridge":{}},"url":"tcp://10.244.1.11:5003","expiration_time":{"seconds":1658407556,"nanos":775507202}}
    Jul 21 12:44:57.086 [TRAC] [type:registry] (1.1)   register={"name":"forwarder-vpp-jk2bt","network_service_names":["forwarder"],"network_service_labels":{"forwarder":{"labels":{"nodeName":"kind-worker2","p2p":"true"}}},"url":"tcp://10.244.2.10:5001","expiration_time":{"seconds":1658407557,"nanos":80750458}}
    ```

- reply

    ```bash
    grep "(1.2)   register-response={\"name\"" registry-k8s-7974cd559f-mbcw6.log
    ```

    Example output:

    ```bash
    Jul 21 12:44:56.772 [TRAC] [type:registry] (1.2)   register-response={"name":"finance-bridge","payload":"ETHERNET"}
    Jul 21 12:44:56.784 [TRAC] [type:registry] (1.2)   register-response={"name":"nse-remote-vlan-66bfccf745-qmx22","network_service_names":["finance-bridge"],"network_service_labels":{"finance-bridge":{}},"url":"tcp://10.244.1.11:5003","expiration_time":{"seconds":1658407556,"nanos":775507202},"initial_registration_time":{"seconds":1658407496,"nanos":779045092}}
    Jul 21 12:44:57.090 [TRAC] [type:registry] (1.2)   register-response={"name":"forwarder-vpp-jk2bt","network_service_names":["forwarder"],"network_service_labels":{"forwarder":{"labels":{"nodeName":"kind-worker2","p2p":"true"}}},"url":"tcp://10.244.2.10:5001","expiration_time":{"seconds":1658407557,"nanos":80750458},"initial_registration_time":{"seconds":1658407497,"nanos":87097752}}
    ```

  Run the following command to inspect the nsmgr logs for;

- request

    ```bash
    grep "(1.1)   register={\"name\"" nsmgr-rpb7r-nsmgr.log
    ```

    Example output:

    ```bash
    Jul 21 12:44:56.570 [TRAC] [type:registry] (1.1)   register={"name":"forwarder-vpp-jk2bt","network_service_names":["forwarder"],"network_service_labels":{"forwarder":{"labels":{"p2p":"true"}}},"url":"inode://2097261/3541380"}
    Jul 21 12:44:57.080 [TRAC] [type:registry] (1.1)   register={"name":"forwarder-vpp-jk2bt","network_service_names":["forwarder"],"network_service_labels":{"forwarder":{"labels":{"p2p":"true"}}},"url":"inode://2097261/3541380"}
    Jul 21 12:45:37.087 [TRAC] [type:registry] (1.1)   register={"name":"forwarder-vpp-jk2bt","network_service_names":["forwarder"],"network_service_labels":{"forwarder":{"labels":{"nodeName":"kind-worker2","p2p":"true"}}},"url":"inode://2097261/3541380","initial_registration_time":{"seconds":1658407497,"nanos":87097752}}
    Jul 21 12:46:17.253 [TRAC] [type:registry] (1.1)   register={"name":"forwarder-vpp-jk2bt","network_service_names":["forwarder"],"network_service_labels":{"forwarder":{"labels":{"nodeName":"kind-worker2","p2p":"true"}}},"url":"inode://2097261/3541380","initial_registration_time":{"seconds":1658407497,"nanos":87097752}}
    ```

- reply

    ```bash
    grep "(1.2)   register-response={\"name\"" nsmgr-rpb7r-nsmgr.log
    ```

    Example output:

    ```bash
    Jul 21 12:44:57.090 [TRAC] [type:registry] (1.2)   register-response={"name":"forwarder-vpp-jk2bt","network_service_names":["forwarder"],"network_service_labels":{"forwarder":{"labels":{"nodeName":"kind-worker2","p2p":"true"}}},"url":"inode://2097261/3541380","expiration_time":{"seconds":1658407557,"nanos":80750458},"initial_registration_time":{"seconds":1658407497,"nanos":87097752}}
    Jul 21 12:45:37.562 [TRAC] [type:registry] (1.2)   register-response={"name":"forwarder-vpp-jk2bt","network_service_names":["forwarder"],"network_service_labels":{"forwarder":{"labels":{"nodeName":"kind-worker2","p2p":"true"}}},"url":"inode://2097261/3541380","expiration_time":{"seconds":1658407597,"nanos":87695905},"initial_registration_time":{"seconds":1658407497,"nanos":87097752}}
    Jul 21 12:46:17.277 [TRAC] [type:registry] (1.2)   register-response={"name":"forwarder-vpp-jk2bt","network_service_names":["forwarder"],"network_service_labels":{"forwarder":{"labels":{"nodeName":"kind-worker2","p2p":"true"}}},"url":"inode://2097261/3541380","expiration_time":{"seconds":1658407637,"nanos":253337453},"initial_registration_time":{"seconds":1658407497,"nanos":87097752}}
    ```

## Connection Setup
  
  These logs are on 'TRACE' level also.
  
  Run the following commands to check forwarder log printouts related to connection setup:

- request

    ```bash
    grep "1)[[:space:]]*request={" forwarder-vpp-jk2bt.log
    ```

    Example output:

    ```bash
    Jul 21 12:45:10.350 [TRAC] [id:0a57021d-d71d-4960-8384-2edc5fb13829] [type:networkService] (1.1)   request={"connection":{"id":"0a57021d-d71d-4960-8384-2edc5fb13829","network_service":"finance-bridge","context":{"ip_context":{"excluded_prefixes":["10.96.0.0/16","10.244.0.0/16"]}},"labels":{"nodeName":"kind-worker2","podName":"iperf1-s-5c446597b8-7c877"},"path":{"index":1,"path_segments":[{"name":"iperf1-s-5c446597b8-7c877","id":"iperf1-s-5c446597b8-7c877-0","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9ucy13dHg4di9wb2QvaXBlcmYxLXMtNWM0NDY1OTdiOC03Yzg3NyIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciJdLCJleHAiOjE2NTg0MDgxMTB9.hxQrZjyXdC8ENAC260YBtcEXE0eLfa2zv6rn7BI30oysigO8LUVgLqztNH7b27XousZJN4UQtdNFMVsPWFQymg","expires":{"seconds":1658408110,"nanos":317091593}},{"name":"nsmgr-rpb7r","id":"0a57021d-d71d-4960-8384-2edc5fb13829","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9ucy13dHg4di9wb2QvaXBlcmYxLXMtNWM0NDY1OTdiOC03Yzg3NyJdLCJleHAiOjE2NTg0MDgxMTB9.1ssFpOnIeKBF92zemvmgzjPjGeCkCbabro38aRRCMhxUg9rVqG_kaXiI8G-hIyyVjDwvnRgEtHLeiQ7-g9Yqtw","expires":{"seconds":1658408110,"nanos":318211336}}]}},"mechanism_preferences":[{"cls":"LOCAL","type":"KERNEL","parameters":{"inodeURL":"inode://4/4026534195","name":"nsm-1"}}]}
    Jul 21 12:45:12.405 [TRAC] [id:0a57021d-d71d-4960-8384-2edc5fb13829] [type:networkService] (1.1)   request={"connection":{"id":"0a57021d-d71d-4960-8384-2edc5fb13829","network_service":"finance-bridge","mechanism":{"cls":"LOCAL","type":"KERNEL","parameters":{"inodeURL":"inode://4/4026534195","name":"nsm-1"}},"context":{"ip_context":{"src_ip_addrs":["172.10.0.2/24","100:200::2/64"],"excluded_prefixes":["10.96.0.0/16","10.244.0.0/16"]},"ethernet_context":{"src_mac":"02:42:ac:14:00:03"}},"labels":{"nodeName":"kind-worker2","podName":"iperf1-s-5c446597b8-7c877","via":"gw1"},"path":{"index":1,"path_segments":[{"name":"iperf1-s-5c446597b8-7c877","id":"iperf1-s-5c446597b8-7c877-0","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9ucy13dHg4di9wb2QvaXBlcmYxLXMtNWM0NDY1OTdiOC03Yzg3NyIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciJdLCJleHAiOjE2NTg0MDgxMTJ9.jghyz9ORaDrAwz7fCgZbBu2cGQunMRDxdOJ6fc0ePebfZ4ZnwUT2WAfjMBwpkhZ6xa5zqraA9AVMhrGDPga6Hg","expires":{"seconds":1658408112,"nanos":393381476}},{"name":"nsmgr-rpb7r","id":"0a57021d-d71d-4960-8384-2edc5fb13829","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9ucy13dHg4di9wb2QvaXBlcmYxLXMtNWM0NDY1OTdiOC03Yzg3NyJdLCJleHAiOjE2NTg0MDgxMTJ9.C-4Q28czLJl_fNuVf31qNKTyhSXZf6d7xyoT-fezwuUUsl0ctnAn1KwRLuHmrUQ8Zh8VXTESCzgnWHFqLi-bQQ","expires":{"seconds":1658408112,"nanos":394204145}},{"name":"forwarder-vpp-jk2bt","id":"5fbfb0e2-7f22-4476-8d13-528f8479c186","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0IiwiYXVkIjpbInNwaWZmZTovL2V4YW1wbGUub3JnL25zL25zbS1zeXN0ZW0vcG9kL25zZS1yZW1vdGUtdmxhbi02NmJmY2NmNzQ1LXFteDIyIl0sImV4cCI6MTY1ODQwODExMH0.dXOR4FX_CC30BU4Uqq429aR62FLC9PDpNBm3rGZQAdKYrBPaKw8AJiEE9yLmaeQXf8Ys0E6uMguYfftX5u8hyQ","expires":{"seconds":1658408110,"nanos":455412760},"metrics":{"client_drops":"0","client_rx_bytes":"0","client_rx_packets":"0","client_tx_bytes":"0","client_tx_packets":"0","server_drops":"0","server_rx_bytes":"0","server_rx_packets":"0","server_tx_bytes":"0","server_tx_packets":"0"}},{"name":"nse-remote-vlan-66bfccf745-qmx22","id":"fe11c2de-3429-463f-80ce-6e673f932c76","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc2UtcmVtb3RlLXZsYW4tNjZiZmNjZjc0NS1xbXgyMiIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0Il0sImV4cCI6MTY1ODQwNzU3MH0.t6TvG7NOsH0Qih7VSieQlO67cEhqc0FCUBZlvt01uoK2GGZ1Tf5t9ouOxSDM5AW6DwnZVQaSFCtaMj0lu_-FnQ","expires":{"seconds":1658407570,"nanos":456917499}}]},"network_service_endpoint_name":"nse-remote-vlan-66bfccf745-qmx22","payload":"ETHERNET"},"mechanism_preferences":[{"cls":"LOCAL","type":"KERNEL","parameters":{"inodeURL":"inode://4/4026534195","name":"nsm-1"}}]}
    Jul 21 12:45:24.484 [TRAC] [id:0a57021d-d71d-4960-8384-2edc5fb13829] [type:networkService] (1.1)   request={"connection":{"id":"0a57021d-d71d-4960-8384-2edc5fb13829","network_service":"finance-bridge","mechanism":{"cls":"LOCAL","type":"KERNEL","parameters":{"inodeURL":"inode://4/4026534195","name":"nsm-1"}},"context":{"ip_context":{"src_ip_addrs":["172.10.0.2/24","100:200::2/64"],"excluded_prefixes":["10.96.0.0/16","10.244.0.0/16"]},"ethernet_context":{"src_mac":"02:42:ac:14:00:03"}},"labels":{"nodeName":"kind-worker2","podName":"iperf1-s-5c446597b8-7c877","via":"gw1"},"path":{"index":1,"path_segments":[{"name":"iperf1-s-5c446597b8-7c877","id":"iperf1-s-5c446597b8-7c877-0","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9ucy13dHg4di9wb2QvaXBlcmYxLXMtNWM0NDY1OTdiOC03Yzg3NyIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciJdLCJleHAiOjE2NTg0MDgxMjR9.rVSMO5gYqiC7sI8Y3YUZM8LmRh5vFDt9WRbqOOX3r80mpsfXzuIw-VVyhYmrF_VHGcDRly4pkV4FDaBf7TE4rQ","expires":{"seconds":1658408124,"nanos":471093485}},{"name":"nsmgr-rpb7r","id":"0a57021d-d71d-4960-8384-2edc5fb13829","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9ucy13dHg4di9wb2QvaXBlcmYxLXMtNWM0NDY1OTdiOC03Yzg3NyJdLCJleHAiOjE2NTg0MDgxMjR9.bf9ER2-l3I18Y1p8On6bVIXNZAfPr7w6AaS3ZOFqM0PcT6l8hr1Nmu_edEufhSGl6iNokTlr0zbrafiFpR5eBw","expires":{"seconds":1658408124,"nanos":471973707}},{"name":"forwarder-vpp-jk2bt","id":"5fbfb0e2-7f22-4476-8d13-528f8479c186","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0IiwiYXVkIjpbInNwaWZmZTovL2V4YW1wbGUub3JnL25zL25zbS1zeXN0ZW0vcG9kL25zZS1yZW1vdGUtdmxhbi02NmJmY2NmNzQ1LXFteDIyIl0sImV4cCI6MTY1ODQwODExMn0.UiuYhu7FzigVQJrlNCpNS-QHt6fry9OP_xsKlLG8AFPOnFm3UKI1fgEzqTXTHs88OJNr_rIdNt_S6saGH-4NkQ","expires":{"seconds":1658408112,"nanos":433748727},"metrics":{"client_drops":"0","client_rx_bytes":"358","client_rx_packets":"3","client_tx_bytes":"546","client_tx_packets":"5","server_drops":"0","server_rx_bytes":"526","server_rx_packets":"5","server_tx_bytes":"346","server_tx_packets":"3"}},{"name":"nse-remote-vlan-66bfccf745-qmx22","id":"fe11c2de-3429-463f-80ce-6e673f932c76","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc2UtcmVtb3RlLXZsYW4tNjZiZmNjZjc0NS1xbXgyMiIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0Il0sImV4cCI6MTY1ODQwNzU3Mn0.8IBh-lDAdPLxNNXLm5NLjHe3PwP69iYEshnjmHzPtNyu1qtbgdDPFjYB4-7xyHLGoRzrC9HIqXPDM28XfPQrrw","expires":{"seconds":1658407572,"nanos":434722147}}]},"network_service_endpoint_name":"nse-remote-vlan-66bfccf745-qmx22","payload":"ETHERNET"},"mechanism_preferences":[{"cls":"LOCAL","type":"KERNEL","parameters":{"inodeURL":"inode://4/4026534195","name":"nsm-1"}}]}
    ```

- reply

    ```bash
    grep "1)[[:space:]]*request-response={" forwarder-vpp-jk2bt.log
    ```

    Example output:

    ```bash
    Jul 21 12:45:10.726 [TRAC] [id:5fbfb0e2-7f22-4476-8d13-528f8479c186] [type:networkService] (77.1)                                                                               request-response={"id":"5fbfb0e2-7f22-4476-8d13-528f8479c186","network_service":"finance-bridge","mechanism":{"cls":"REMOTE","type":"VLAN","parameters":{"vlan-id":"100"}},"context":{"ip_context":{"src_ip_addrs":["172.10.0.2/24","100:200::2/64"],"excluded_prefixes":["10.96.0.0/16","10.244.0.0/16"]}},"labels":{"via":"gw1"},"path":{"index":2,"path_segments":[{"name":"iperf1-s-5c446597b8-7c877","id":"iperf1-s-5c446597b8-7c877-0","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9ucy13dHg4di9wb2QvaXBlcmYxLXMtNWM0NDY1OTdiOC03Yzg3NyIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciJdLCJleHAiOjE2NTg0MDgxMTB9.hxQrZjyXdC8ENAC260YBtcEXE0eLfa2zv6rn7BI30oysigO8LUVgLqztNH7b27XousZJN4UQtdNFMVsPWFQymg","expires":{"seconds":1658408110,"nanos":317091593}},{"name":"nsmgr-rpb7r","id":"0a57021d-d71d-4960-8384-2edc5fb13829","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0Il0sImV4cCI6MTY1ODQwODExMH0.UDmBvbxJgImztuUkdU4-4w7MhN1UtJwe6Rgp2cRenggCDdfWpVTy2MmPL-WYIEYDy4SK4SYHRi-A3ogGDn7vHA","expires":{"seconds":1658408110,"nanos":350078099}},{"name":"forwarder-vpp-jk2bt","id":"5fbfb0e2-7f22-4476-8d13-528f8479c186","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0IiwiYXVkIjpbInNwaWZmZTovL2V4YW1wbGUub3JnL25zL25zbS1zeXN0ZW0vcG9kL25zZS1yZW1vdGUtdmxhbi02NmJmY2NmNzQ1LXFteDIyIl0sImV4cCI6MTY1ODQwODExMH0.dXOR4FX_CC30BU4Uqq429aR62FLC9PDpNBm3rGZQAdKYrBPaKw8AJiEE9yLmaeQXf8Ys0E6uMguYfftX5u8hyQ","expires":{"seconds":1658408110,"nanos":455412760}},{"name":"nse-remote-vlan-66bfccf745-qmx22","id":"fe11c2de-3429-463f-80ce-6e673f932c76","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc2UtcmVtb3RlLXZsYW4tNjZiZmNjZjc0NS1xbXgyMiIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0Il0sImV4cCI6MTY1ODQwNzU3MH0.t6TvG7NOsH0Qih7VSieQlO67cEhqc0FCUBZlvt01uoK2GGZ1Tf5t9ouOxSDM5AW6DwnZVQaSFCtaMj0lu_-FnQ","expires":{"seconds":1658407570,"nanos":456917499}}]},"network_service_endpoint_name":"nse-remote-vlan-66bfccf745-qmx22","payload":"ETHERNET"}
    Jul 21 12:45:12.441 [TRAC] [id:5fbfb0e2-7f22-4476-8d13-528f8479c186] [type:networkService] (77.1)                                                                               request-response={"id":"5fbfb0e2-7f22-4476-8d13-528f8479c186","network_service":"finance-bridge","mechanism":{"cls":"REMOTE","type":"VLAN","parameters":{"vlan-id":"100"}},"context":{"ip_context":{"src_ip_addrs":["172.10.0.2/24","100:200::2/64"],"excluded_prefixes":["10.96.0.0/16","10.244.0.0/16"]},"ethernet_context":{"src_mac":"02:42:ac:14:00:03"}},"labels":{"via":"gw1"},"path":{"index":2,"path_segments":[{"name":"iperf1-s-5c446597b8-7c877","id":"iperf1-s-5c446597b8-7c877-0","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9ucy13dHg4di9wb2QvaXBlcmYxLXMtNWM0NDY1OTdiOC03Yzg3NyIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciJdLCJleHAiOjE2NTg0MDgxMTJ9.jghyz9ORaDrAwz7fCgZbBu2cGQunMRDxdOJ6fc0ePebfZ4ZnwUT2WAfjMBwpkhZ6xa5zqraA9AVMhrGDPga6Hg","expires":{"seconds":1658408112,"nanos":393381476}},{"name":"nsmgr-rpb7r","id":"0a57021d-d71d-4960-8384-2edc5fb13829","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0Il0sImV4cCI6MTY1ODQwODExMn0.5KsXKeXcJa_678-8BvDRJrs8hKzU07EqWeXGEO4JFmjgXOaJSCVS1NJr9dBuVj1yQ4X-y8ktbdYRKxk93WqWTg","expires":{"seconds":1658408112,"nanos":405118549}},{"name":"forwarder-vpp-jk2bt","id":"5fbfb0e2-7f22-4476-8d13-528f8479c186","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0IiwiYXVkIjpbInNwaWZmZTovL2V4YW1wbGUub3JnL25zL25zbS1zeXN0ZW0vcG9kL25zZS1yZW1vdGUtdmxhbi02NmJmY2NmNzQ1LXFteDIyIl0sImV4cCI6MTY1ODQwODExMn0.UiuYhu7FzigVQJrlNCpNS-QHt6fry9OP_xsKlLG8AFPOnFm3UKI1fgEzqTXTHs88OJNr_rIdNt_S6saGH-4NkQ","expires":{"seconds":1658408112,"nanos":433748727},"metrics":{"client_drops":"0","client_rx_bytes":"0","client_rx_packets":"0","client_tx_bytes":"0","client_tx_packets":"0","server_drops":"0","server_rx_bytes":"0","server_rx_packets":"0","server_tx_bytes":"0","server_tx_packets":"0"}},{"name":"nse-remote-vlan-66bfccf745-qmx22","id":"fe11c2de-3429-463f-80ce-6e673f932c76","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc2UtcmVtb3RlLXZsYW4tNjZiZmNjZjc0NS1xbXgyMiIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0Il0sImV4cCI6MTY1ODQwNzU3Mn0.8IBh-lDAdPLxNNXLm5NLjHe3PwP69iYEshnjmHzPtNyu1qtbgdDPFjYB4-7xyHLGoRzrC9HIqXPDM28XfPQrrw","expires":{"seconds":1658407572,"nanos":434722147}}]},"network_service_endpoint_name":"nse-remote-vlan-66bfccf745-qmx22","payload":"ETHERNET"}
    Jul 21 12:45:24.532 [TRAC] [id:5fbfb0e2-7f22-4476-8d13-528f8479c186] [type:networkService] (77.1)                                                                               request-response={"id":"5fbfb0e2-7f22-4476-8d13-528f8479c186","network_service":"finance-bridge","mechanism":{"cls":"REMOTE","type":"VLAN","parameters":{"vlan-id":"100"}},"context":{"ip_context":{"src_ip_addrs":["172.10.0.2/24","100:200::2/64"],"excluded_prefixes":["10.96.0.0/16","10.244.0.0/16"]},"ethernet_context":{"src_mac":"02:42:ac:14:00:03"}},"labels":{"via":"gw1"},"path":{"index":2,"path_segments":[{"name":"iperf1-s-5c446597b8-7c877","id":"iperf1-s-5c446597b8-7c877-0","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9ucy13dHg4di9wb2QvaXBlcmYxLXMtNWM0NDY1OTdiOC03Yzg3NyIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciJdLCJleHAiOjE2NTg0MDgxMjR9.rVSMO5gYqiC7sI8Y3YUZM8LmRh5vFDt9WRbqOOX3r80mpsfXzuIw-VVyhYmrF_VHGcDRly4pkV4FDaBf7TE4rQ","expires":{"seconds":1658408124,"nanos":471093485}},{"name":"nsmgr-rpb7r","id":"0a57021d-d71d-4960-8384-2edc5fb13829","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc21nci1ycGI3ciIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0Il0sImV4cCI6MTY1ODQwODEyNH0.3Df9v7evNgUhjBvNHHRpjacCcx5PVPoGcAp4oKaynVYkDkRrzzA2prKvXg711Yn8V4g-_GgkdO9wNxz6InIUPg","expires":{"seconds":1658408124,"nanos":484076688}},{"name":"forwarder-vpp-jk2bt","id":"5fbfb0e2-7f22-4476-8d13-528f8479c186","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0IiwiYXVkIjpbInNwaWZmZTovL2V4YW1wbGUub3JnL25zL25zbS1zeXN0ZW0vcG9kL25zZS1yZW1vdGUtdmxhbi02NmJmY2NmNzQ1LXFteDIyIl0sImV4cCI6MTY1ODQwODEyNH0.kY3ZePSBUmMVh-YJ9eZPSrHlkKC2_7_63c9yZ4V_XUW0zwTJ-loU3BpqINun_AGi_agsD2eaw-DJs-olJCyXqA","expires":{"seconds":1658408124,"nanos":526530867},"metrics":{"client_drops":"0","client_rx_bytes":"358","client_rx_packets":"3","client_tx_bytes":"546","client_tx_packets":"5","server_drops":"0","server_rx_bytes":"526","server_rx_packets":"5","server_tx_bytes":"346","server_tx_packets":"3"}},{"name":"nse-remote-vlan-66bfccf745-qmx22","id":"fe11c2de-3429-463f-80ce-6e673f932c76","token":"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9uc2UtcmVtb3RlLXZsYW4tNjZiZmNjZjc0NS1xbXgyMiIsImF1ZCI6WyJzcGlmZmU6Ly9leGFtcGxlLm9yZy9ucy9uc20tc3lzdGVtL3BvZC9mb3J3YXJkZXItdnBwLWprMmJ0Il0sImV4cCI6MTY1ODQwNzU4NH0.EIN1CmITgoVwgCKbzKCsy7_pyEHSub0CAVU3IJvP75758WZ9YoMH9QoQpb08cpFRyc7APVRw2Q2367EoknUwFg","expires":{"seconds":1658407584,"nanos":527710915}}]},"network_service_endpoint_name":"nse-remote-vlan-66bfccf745-qmx22","payload":"ETHERNET"}
    ```
