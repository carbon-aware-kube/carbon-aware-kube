{{/*
Expand the name of the chart.
*/}}
{{- define "carbon-aware-kube.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "carbon-aware-kube.fullname" -}}
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
{{- define "carbon-aware-kube.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "carbon-aware-kube.labels" -}}
helm.sh/chart: {{ include "carbon-aware-kube.chart" . }}
{{ include "carbon-aware-kube.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "carbon-aware-kube.selectorLabels" -}}
app.kubernetes.io/name: {{ include "carbon-aware-kube.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "carbon-aware-kube.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "carbon-aware-kube.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Determine the scheduler URL to use
*/}}
{{- define "carbon-aware-kube.schedulerUrl" -}}
{{- if .Values.scheduler.local.enabled }}
{{- printf "http://%s-carbon-aware-scheduler:8080" .Release.Name }}
{{- else }}
{{- required "External scheduler URL must be provided when local scheduler is disabled" .Values.scheduler.external.url }}
{{- end }}
{{- end }}
