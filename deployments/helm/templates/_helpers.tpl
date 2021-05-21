{{/* vim: set filetype=mustache: */}}

{{/*
Set IP Family
*/}}
{{- define "meridio.vips" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- (printf "%s,%s" .Values.vipIPv4 .Values.vipIPv6) | quote -}}
{{- else if eq .Values.ipFamily "ipv6" -}}
{{- printf .Values.vipIPv6 -}}
{{- else -}}
{{- printf .Values.vipIPv4 -}}
{{- end -}}
{{- end -}}

{{- define "meridio.subnetPools" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- (printf "%s,%s" .Values.subnetPoolIPv4 .Values.subnetPoolIPv6) | quote -}}
{{- else if eq .Values.ipFamily "ipv6" -}}
{{- printf .Values.subnetPoolIPv6 -}}
{{- else -}}
{{- printf .Values.subnetPoolIPv4 -}}
{{- end -}}
{{- end -}}

{{- define "meridio.subnetPrefixLengths" -}}
{{- if eq .Values.ipFamily "dualstack" -}}
{{- (printf "%d,%d" (int64 .Values.subnetPrefixLengthIPv4) (int64 .Values.subnetPrefixLengthIPv6)) | quote -}}
{{- else if eq .Values.ipFamily "ipv6" -}}
{{- .Values.subnetPrefixLengthIPv6 | quote -}}
{{- else -}}
{{- .Values.subnetPrefixLengthIPv4 | quote -}}
{{- end -}}
{{- end -}}

{{- define "meridio.proxyNetworkServiceName" -}}
{{- printf "%s.%s" .Values.proxyNetworkServiceName .Release.Namespace -}}
{{- end -}}

{{- define "meridio.loadBalancerNetworkServiceName" -}}
{{- printf "%s.%s" .Values.loadBalancerNetworkServiceName .Release.Namespace -}}
{{- end -}}

{{- define "meridio.vlanServiceName" -}}
{{- printf "%s.%s" .Values.vlanServiceName .Release.Namespace -}}
{{- end -}}
