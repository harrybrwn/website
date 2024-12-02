{{/*
*/}}
{{- define "wordpress.wp-install" -}}
{{ if .url -}}
 {{ .url | quote }}
{{- else -}}
{{ .name }}{{ if .version }} --version={{ .version | quote }}{{- end }}
{{- end }}
{{- if .activate }} --activate{{ end }}
{{- end }}

{{/*
*/}}
{{- define "wordpress.wp-plugin-install" -}}
wp plugin install {{ include "wordpress.wp-install" . }}
{{- end }}

{{/*
*/}}
{{- define "wordpress.wp-theme-install" -}}
wp theme install {{ include "wordpress.wp-install" . }}
{{- end }}

{{- /* vim: ft=gotmpl */}}
