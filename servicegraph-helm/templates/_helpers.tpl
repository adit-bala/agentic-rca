{{/*
Expand the name of the chart.
*/}}
{{- define "servicegraph.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "servicegraph.fullname" -}}
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
{{- define "servicegraph.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "servicegraph.labels" -}}
helm.sh/chart: {{ include "servicegraph.chart" . }}
{{ include "servicegraph.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "servicegraph.selectorLabels" -}}
app.kubernetes.io/name: {{ include "servicegraph.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Beyla labels
*/}}
{{- define "servicegraph.beyla.labels" -}}
{{ include "servicegraph.labels" . }}
app.kubernetes.io/component: beyla
{{- end }}

{{/*
Beyla selector labels
*/}}
{{- define "servicegraph.beyla.selectorLabels" -}}
{{ include "servicegraph.selectorLabels" . }}
app.kubernetes.io/component: beyla
{{- end }}

{{/*
OTel Collector labels
*/}}
{{- define "servicegraph.otelCollector.labels" -}}
{{ include "servicegraph.labels" . }}
app.kubernetes.io/component: otel-collector
{{- end }}

{{/*
OTel Collector selector labels
*/}}
{{- define "servicegraph.otelCollector.selectorLabels" -}}
{{ include "servicegraph.selectorLabels" . }}
app.kubernetes.io/component: otel-collector
{{- end }}

{{/*
Namespace
*/}}
{{- define "servicegraph.namespace" -}}
{{- default .Release.Namespace .Values.global.namespace }}
{{- end }}

{{/*
Beyla service account name
*/}}
{{- define "servicegraph.beyla.serviceAccountName" -}}
{{- printf "%s-beyla" (include "servicegraph.fullname" .) }}
{{- end }}

{{/*
OTel Collector service name
*/}}
{{- define "servicegraph.otelCollector.serviceName" -}}
{{- printf "%s-otel-collector" (include "servicegraph.fullname" .) }}
{{- end }}
