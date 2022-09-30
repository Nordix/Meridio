{{/* vim: set filetype=mustache: */}}

{{/*
Set IP Family
*/}}

{{- define "meridio.loadBalancer.sysctls" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- printf "sysctl -w net.ipv6.conf.all.forwarding=1 ; sysctl -w net.ipv4.conf.all.forwarding=1 ; sysctl -w net.ipv4.fib_multipath_hash_policy=1 ; sysctl -w net.ipv6.fib_multipath_hash_policy=1 ; sysctl -w net.ipv4.conf.all.rp_filter=0 ; sysctl -w net.ipv4.conf.default.rp_filter=0" -}}
{{- else if eq .Values.ipFamily "ipv6" -}}
{{- printf "sysctl -w net.ipv6.conf.all.forwarding=1 ; sysctl -w net.ipv6.fib_multipath_hash_policy=1" -}}
{{- else -}}
{{- printf "sysctl -w net.ipv4.conf.all.forwarding=1 ; sysctl -w net.ipv4.fib_multipath_hash_policy=1" -}}
{{- end -}}
{{- end -}}

{{- define "meridio.proxy.sysctls" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- printf "sysctl -w net.ipv6.conf.all.forwarding=1 ; sysctl -w net.ipv4.conf.all.forwarding=1 ; sysctl -w net.ipv6.conf.all.accept_dad=0 ; sysctl -w net.ipv4.fib_multipath_hash_policy=1 ; sysctl -w net.ipv6.fib_multipath_hash_policy=1 ; sysctl -w net.ipv4.conf.all.rp_filter=0 ; sysctl -w net.ipv4.conf.default.rp_filter=0" -}}
{{- else if eq .Values.ipFamily "ipv6" -}}
{{- printf "sysctl -w net.ipv6.conf.all.forwarding=1 ; sysctl -w net.ipv6.conf.all.accept_dad=0 ; sysctl -w net.ipv6.fib_multipath_hash_policy=1" -}}
{{- else -}}
{{- printf "sysctl -w net.ipv4.conf.all.forwarding=1 ; sysctl -w net.ipv4.fib_multipath_hash_policy=1" -}}
{{- end -}}
{{- end -}}

{{- define "meridio.nsp.serviceName" -}}
{{- printf "%s-%s" .Values.nsp.serviceName .Values.trench.name -}}
{{- end -}}

{{- define "meridio.ipam.serviceName" -}}
{{- printf "%s-%s" .Values.ipam.serviceName .Values.trench.name -}}
{{- end -}}

{{- define "meridio.proxy.networkServiceName" -}}
{{- printf "%s.%s.%s" .Values.proxy.networkServiceName .Values.trench.name .Release.Namespace -}}
{{- end -}}

{{- define "meridio.loadBalancer.networkServiceName" -}}
{{- printf "%s.%s.%s" .Values.loadBalancer.networkServiceName .Values.trench.name .Release.Namespace -}}
{{- end -}}

{{- define "meridio.vlan.networkServiceName" -}}
{{- printf "%s.%s.%s" .Values.vlan.networkServiceName .Values.trench.name .Release.Namespace -}}
{{- end -}}

{{- define "meridio.vlan.extInterfaceName" -}}
{{- if .Values.vlan.id }}
{{- printf "ext-vlan.%d" ( .Values.vlan.id | int ) -}}
{{- else -}}
{{- printf "ext" -}}
{{- end -}}
{{- end -}}

{{- define "meridio.vrrps" -}}
{{- join "," .Values.vlan.fe.vrrp }}
{{- end -}}

{{- define "meridio.configuration" -}}
{{- printf "%s-%s" .Values.configuration.configmap .Values.trench.name -}}
{{- end -}}

{{- define "meridio.serviceAccount" -}}
{{- printf "meridio-%s" .Values.trench.name -}}
{{- end -}}

{{- define "meridio.authServiceAccount" -}}
{{- printf "meridio-auth-%s" .Values.trench.name -}}
{{- end -}}

{{- define "meridio.startupProbe" -}}
{{- $healthAddr := .root.Values.probe.addr -}}
{{- $healthService := .root.Values.probe.service -}}
{{- $spiffe := false -}}
{{- if .component.probe -}}
{{- $healthAddr = .component.probe.addr | default $healthAddr -}}
{{- $healthService = .component.probe.service | default $healthService -}}
{{- $spiffe = .component.probe.spiffe | default $spiffe -}}
{{- if .component.probe.startup }}
{{- with .component.probe.startup -}}
{{- $healthAddr = .addr | default $healthAddr -}}
{{- $healthService = .service | default $healthService -}}
{{- $spiffe = .spiffe | default $spiffe -}}
{{- end -}}
{{- end -}}
{{- end -}}
exec:
  command:
  - /bin/grpc_health_probe
{{- if $spiffe }}
  - -spiffe
{{- end }}
  - -addr={{ $healthAddr }}
  - -service={{ $healthService }}
  - -connect-timeout=100ms
  - -rpc-timeout=150ms
initialDelaySeconds: 0
periodSeconds: 2
timeoutSeconds: 2
failureThreshold: 30
{{- end -}}

{{- define "meridio.livenessProbe" -}}
{{- $healthAddr := .root.Values.probe.addr -}}
{{- $healthService := .root.Values.probe.service -}}
{{- $spiffe := false -}}
{{- if .component.probe -}}
{{- $healthAddr = .component.probe.addr | default $healthAddr -}}
{{- $healthService = .component.probe.service | default $healthService -}}
{{- $spiffe = .component.probe.spiffe | default $spiffe -}}
{{- if .component.probe.liveness }}
{{- with .component.probe.liveness -}}
{{- $healthAddr = .addr | default $healthAddr -}}
{{- $healthService = .service | default $healthService -}}
{{- $spiffe = .spiffe | default $spiffe -}}
{{- end -}}
{{- end -}}
{{- end -}}
exec:
  command:
  - /bin/grpc_health_probe
{{- if $spiffe }}
  - -spiffe
{{- end }}
  - -addr={{ $healthAddr }}
  - -service={{ $healthService }}
  - -connect-timeout=100ms
  - -rpc-timeout=150ms
initialDelaySeconds: 0
periodSeconds: 10
timeoutSeconds: 3
failureThreshold: 5
{{- end -}}

{{- define "meridio.readinessProbe" -}}
{{- $healthAddr := .root.Values.probe.addr -}}
{{- $healthService := .root.Values.probe.service -}}
{{- $spiffe := false -}}
{{- if .component.probe }}
{{- $healthAddr = .component.probe.addr | default $healthAddr -}}
{{- $healthService = .component.probe.service | default $healthService -}}
{{- $spiffe = .component.probe.spiffe | default $spiffe -}}
{{- if .component.probe.readiness }}
{{- with .component.probe.readiness -}}
{{- $healthAddr = .addr | default $healthAddr -}}
{{- $healthService = .service | default $healthService -}}
{{- $spiffe = .spiffe | default $spiffe -}}
{{- end -}}
{{- end -}}
{{- end -}}
exec:
  command:
  - /bin/grpc_health_probe
{{- if $spiffe }}
  - -spiffe
{{- end }}
  - -addr={{ $healthAddr }}
  - -service={{ $healthService }}
  - -connect-timeout=100ms
  - -rpc-timeout=150ms
initialDelaySeconds: 0
periodSeconds: 10
timeoutSeconds: 3
failureThreshold: 5
{{- end -}}

{{- define "meridio.bgpAuth" -}}
{{- if .component.bgpAuth -}}
{{- if .component.bgpAuth.bgpAuthKey -}}
{{- if .component.bgpAuth.bgpAuthKeySource -}}
key: {{ .component.bgpAuth.bgpAuthKey }}
source: {{ .component.bgpAuth.bgpAuthKeySource }}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
