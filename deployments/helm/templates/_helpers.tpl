{{/* vim: set filetype=mustache: */}}

{{/*
Set IP Family
*/}}
{{- define "meridio.vips" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- (printf "%s,%s" .Values.vip.ipv4 .Values.vip.ipv6) -}}
{{- else if eq .Values.ipFamily "ipv6" -}}
{{- printf .Values.vip.ipv6 -}}
{{- else -}}
{{- printf .Values.vip.ipv4 -}}
{{- end -}}
{{- end -}}

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

{{- define "meridio.proxy.networkServiceName" -}}
{{- printf "%s.%s" .Values.proxy.networkServiceName .Release.Namespace -}}
{{- end -}}

{{- define "meridio.loadBalancer.networkServiceName" -}}
{{- printf "%s.%s" .Values.loadBalancer.networkServiceName .Release.Namespace -}}
{{- end -}}

{{- define "meridio.vlan.networkServiceName" -}}
{{- printf "%s.%s" .Values.vlan.networkServiceName .Release.Namespace -}}
{{- end -}}
