{{/* vim: set filetype=mustache: */}}

{{/*
Set IP Family
*/}}

{{- define "meridio.subnetPool.prefixes" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- (printf "%s,%s" .Values.subnetPool.ipv4 .Values.subnetPool.ipv6) | quote -}}
{{- else if eq .Values.ipFamily "ipv6" -}}
{{- printf .Values.subnetPool.ipv6 -}}
{{- else -}}
{{- printf .Values.subnetPool.ipv4 -}}
{{- end -}}
{{- end -}}

{{- define "meridio.subnetPool.prefixLengths" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- (printf "%d,%d" (int64 .Values.subnetPool.prefixLength.ipv4) (int64 .Values.subnetPool.prefixLength.ipv6)) | quote -}}
{{- else if eq .Values.ipFamily "ipv6" -}}
{{- .Values.subnetPool.prefixLength.ipv6 | quote -}}
{{- else -}}
{{- .Values.subnetPool.prefixLength.ipv4 | quote -}}
{{- end -}}
{{- end -}}

{{- define "meridio.loadBalancer.sysctls" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- printf "sysctl -w net.ipv6.conf.all.forwarding=1 ; sysctl -w net.ipv4.conf.all.forwarding=1 ; sysctl -w net.ipv4.fib_multipath_hash_policy=1 ; sysctl -w net.ipv6.fib_multipath_hash_policy=1" -}}
{{- else if eq .Values.ipFamily "ipv6" -}}
{{- printf "sysctl -w net.ipv6.conf.all.forwarding=1 ; sysctl -w net.ipv6.fib_multipath_hash_policy=1" -}}
{{- else -}}
{{- printf "sysctl -w net.ipv4.conf.all.forwarding=1 ; sysctl -w net.ipv4.fib_multipath_hash_policy=1" -}}
{{- end -}}
{{- end -}}

{{- define "meridio.proxy.sysctls" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- printf "sysctl -w net.ipv6.conf.all.forwarding=1 ; sysctl -w net.ipv4.conf.all.forwarding=1 ; sysctl -w net.ipv6.conf.all.accept_dad=0 ; sysctl -w net.ipv4.fib_multipath_hash_policy=1 ; sysctl -w net.ipv6.fib_multipath_hash_policy=1" -}}
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

{{- define "meridio.vrrps" -}}
{{- join "," .Values.vlan.fe.vrrp }}
{{- end -}}

{{- define "meridio.configuration" -}}
{{- printf "%s-%s" .Values.configuration.configmap .Values.trench.name -}}
{{- end -}}

{{- define "meridio.serviceAccount" -}}
{{- printf "meridio-%s" .Values.trench.name -}}
{{- end -}}
