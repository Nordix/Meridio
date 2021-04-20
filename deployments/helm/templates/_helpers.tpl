{{/* vim: set filetype=mustache: */}}

{{/*
Set IP Family
*/}}
{{- define "meridio.vip" -}}
{{- if eq .Values.ipFamily "ipv6" -}}
{{- printf .Values.vipIPv6 -}}
{{- else -}}
{{- printf .Values.vipIPv4 -}}
{{- end -}}
{{- end -}}

{{- define "meridio.subnetPool" -}}
{{- if eq .Values.ipFamily "ipv6" -}}
{{- printf .Values.subnetPoolIPv6 -}}
{{- else -}}
{{- printf .Values.subnetPoolIPv4 -}}
{{- end -}}
{{- end -}}

{{- define "meridio.subnetPrefixLength" -}}
{{- if eq .Values.ipFamily "ipv6" -}}
{{- .Values.subnetPrefixLengthIPv6 | quote -}}
{{- else -}}
{{- .Values.subnetPrefixLengthIPv4 | quote -}}
{{- end -}}
{{- end -}}

{{- define "meridio.vlanPrefix" -}}
{{- if eq .Values.ipFamily "ipv6" -}}
{{- .Values.vlanIPv6Prefix | quote -}}
{{- else -}}
{{- .Values.vlanIPv4Prefix | quote -}}
{{- end -}}
{{- end -}}

{{- define "meridio.proxyNetworkServiceName" -}}
{{- printf "%s.%s" .Values.proxyNetworkServiceName .Release.Namespace -}}
{{- end -}}

{{- define "meridio.loadBalancerNetworkServiceName" -}}
{{- printf "%s.%s" .Values.loadBalancerNetworkServiceName .Release.Namespace -}}
{{- end -}}

