{{/*
Copyright 2025 Tencent
SPDX-License-Identifier: MIT

Rethra Helm Chart Template Helpers

Best Practices References:
- https://helm.sh/docs/chart_best_practices/templates/
- https://github.com/argoproj/argo-helm/blob/main/charts/argo-cd/templates/_helpers.tpl
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "rethra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "rethra.fullname" -}}
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
Ref: https://helm.sh/docs/chart_best_practices/labels/
*/}}
{{- define "rethra.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels following Kubernetes recommended labels.
Ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
*/}}
{{- define "rethra.labels" -}}
helm.sh/chart: {{ include "rethra.chart" . }}
{{ include "rethra.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: rethra
{{- end }}

{{/*
Selector labels
*/}}
{{- define "rethra.selectorLabels" -}}
app.kubernetes.io/name: {{ include "rethra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Component labels - use for individual components
Usage: {{ include "rethra.componentLabels" (dict "component" "app" "context" .) }}
*/}}
{{- define "rethra.componentLabels" -}}
{{ include "rethra.labels" .context }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Component selector labels
Usage: {{ include "rethra.componentSelectorLabels" (dict "component" "app" "context" .) }}
*/}}
{{- define "rethra.componentSelectorLabels" -}}
{{ include "rethra.selectorLabels" .context }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Create the name of the service account to use.
Ref: https://helm.sh/docs/chart_best_practices/rbac/
*/}}
{{- define "rethra.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "rethra.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Secret name - supports existing secret
*/}}
{{- define "rethra.secretName" -}}
{{- if .Values.secrets.existingSecret }}
{{- .Values.secrets.existingSecret }}
{{- else }}
{{- include "rethra.fullname" . }}-secrets
{{- end }}
{{- end }}

{{/*
Return the app image with tag.
Defaults to Chart.appVersion if tag is not specified.
*/}}
{{- define "rethra.app.image" -}}
{{- $tag := default .Chart.AppVersion .Values.app.image.tag }}
{{- printf "%s:%s" .Values.app.image.repository $tag }}
{{- end }}

{{/*
Return the frontend image with tag.
Defaults to Chart.appVersion if tag is not specified.
*/}}
{{- define "rethra.frontend.image" -}}
{{- $tag := default .Chart.AppVersion .Values.frontend.image.tag }}
{{- printf "%s:%s" .Values.frontend.image.repository $tag }}
{{- end }}

{{/*
Return the docreader image with tag.
Defaults to Chart.appVersion if tag is not specified.
*/}}
{{- define "rethra.docreader.image" -}}
{{- $tag := default .Chart.AppVersion .Values.docreader.image.tag }}
{{- printf "%s:%s" .Values.docreader.image.repository $tag }}
{{- end }}

{{/*
Return the PostgreSQL image with tag.
*/}}
{{- define "rethra.postgresql.image" -}}
{{- printf "%s:%s" .Values.postgresql.image.repository .Values.postgresql.image.tag }}
{{- end }}

{{/*
Return the Redis image with tag.
*/}}
{{- define "rethra.redis.image" -}}
{{- printf "%s:%s" .Values.redis.image.repository .Values.redis.image.tag }}
{{- end }}

{{/*
Return the Neo4j image with tag.
*/}}
{{- define "rethra.neo4j.image" -}}
{{- printf "%s:%s" .Values.neo4j.image.repository .Values.neo4j.image.tag }}
{{- end }}

{{/*
Create image pull secrets list.
*/}}
{{- define "rethra.imagePullSecrets" -}}
{{- with .Values.global.imagePullSecrets }}
imagePullSecrets:
{{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Return the storage class name.
*/}}
{{- define "rethra.storageClass" -}}
{{- if .Values.global.storageClass }}
{{- if eq .Values.global.storageClass "-" }}
storageClassName: ""
{{- else }}
storageClassName: {{ .Values.global.storageClass | quote }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Pod security context.
Merges global defaults with component-specific overrides.
*/}}
{{- define "rethra.podSecurityContext" -}}
{{- $global := .Values.global.podSecurityContext | default dict }}
{{- $component := .componentSecurityContext | default dict }}
{{- $merged := merge $component $global }}
{{- if $merged }}
securityContext:
{{- toYaml $merged | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Container security context.
*/}}
{{- define "rethra.containerSecurityContext" -}}
{{- if . }}
securityContext:
{{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}
