{{- define "gw.vlanConf" -}}
{{- if .Values.postfix -}}
{{- printf "vlan-conf-%s" .Values.postfix -}}
{{ else }}
{{- printf "vlan-conf" -}}
{{- end -}}
{{- end -}}

{{- define "gw.gwtgConf" -}}
{{- if .Values.postfix -}}
{{- printf "gwtg-conf-%s" .Values.postfix -}}
{{ else }}
{{- printf "gwtg-conf" -}}
{{- end -}}
{{- end -}}

{{- define "gw.gwLabel" -}}
{{- if .Values.postfix -}}
{{- printf "gateway-%s" .Values.postfix -}}
{{ else }}
{{- printf "gateway" -}}
{{- end -}}
{{- end -}}

{{- define "gw.gw1" -}}
{{- if .Values.postfix -}}
{{- printf "gateway-1-%s" .Values.postfix -}}
{{ else }}
{{- printf "gateway-1" -}}
{{- end -}}
{{- end -}}

{{- define "gw.gw2" -}}
{{- if .Values.postfix -}}
{{- printf "gateway-2-%s" .Values.postfix -}}
{{ else }}
{{- printf "gateway-2" -}}
{{- end -}}
{{- end -}}

{{- define "tg.tgConf" -}}
{{- if .Values.postfix -}}
{{- printf "tg-conf-%s" .Values.postfix -}}
{{ else }}
{{- printf "tg-conf" -}}
{{- end -}}
{{- end -}}

{{- define "tg.tg" -}}
{{- if .Values.postfix -}}
{{- printf "tg-%s" .Values.postfix -}}
{{ else }}
{{- printf "tg" -}}
{{- end -}}
{{- end -}}
