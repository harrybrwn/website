{{/*
Expand the name of the chart.
*/}}
{{- define "wordpress.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "wordpress.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "wordpress.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels that can be applied to any object.
*/}}
{{- define "wordpress.commonLabels" -}}
helm.sh/chart: {{ include "wordpress.chart" . }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "wordpress.labels" -}}
{{ include "wordpress.commonLabels" . }}
{{ include "wordpress.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ include "wordpress.fullname" . }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "wordpress.selectorLabels" -}}
app.kubernetes.io/name: {{ include "wordpress.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
*/}}
{{- define "wordpress.imagePullSecrets" -}}
{{- if .Values.imagePullSecrets -}}
imagePullSecrets:
  {{- toYaml .Values.imagePullSecrets | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "wordpress.serviceAccountName" -}}
{{- if .Values.wordpress.serviceAccount.create }}
{{- default (include "wordpress.fullname" .) .Values.wordpress.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.wordpress.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
The full wordpress-runtime container image.
*/}}
{{- define "wordpress.image" -}}
{{ .Values.wordpress.image.repository }}:{{ .Values.wordpress.image.tag | default .Chart.AppVersion }}
{{- end }}

{{/*
Name of the secret that the certificate is stored in.
*/}}
{{- define "wordpress.certSecret" -}}
{{- if and .Values.cert .Values.cert.existingSecretName }}
{{- .Values.cert.existingSecretName }}
{{- else }}
{{- include "wordpress.fullname" . }}-tls
{{- end }}
{{- end }}

{{/*
Returns the pathPrefix that will be used by the wordpress-operator.
*/}}
{{- define "wordpress.pathPrefix" -}}
{{- if .Values.wordpress.pathPrefix }}
  {{- if isAbs .Values.wordpress.pathPrefix }}
    {{- .Values.wordpress.pathPrefix }}
  {{- else -}}
    /{{ .Values.wordpress.pathPrefix }}
  {{- end }}
{{- else -}}
/wp
{{- end }}
{{- end }}

{{/*
*/}}
{{- define "wordpress.wp_home" -}}
{{- if or .Values.cert.create .Values.cert.existingSecretName -}}
https
{{- else -}}
http
{{- end }}://{{ .Values.domain }}
{{- end }}

{{/* vim: ft=gotmpl */}}
