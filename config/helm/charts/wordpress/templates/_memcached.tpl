{{/*
Memcached name
*/}}
{{- define "wp.memcached.name" -}}
{{ include "wordpress.fullname" . }}-memcached
{{- end }}

{{/*
Memcached selector labels
*/}}
{{- define "wp.memcached.selectorLabels" -}}
app.kubernetes.io/name: memcached
app.kubernetes.io/component: cache
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Values.memcached.image.tag }}
app.kubernetes.io/version: {{ .Values.memcached.image.tag }}
{{- end }}
app.kubernetes.io/part-of: {{ include "wordpress.fullname" . }}
{{- end }}

{{/*
Memcached Labels
*/}}
{{- define "wp.memcached.labels" -}}
{{ include "wp.memcached.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{ include "wordpress.commonLabels" . }}
{{- end }}
{{/* vim: ft=gotmpl ts=2 sw=2 */}}
