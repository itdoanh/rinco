{{/*
RINCO Helm chart — helper templates
*/}}

{{/* ============================================================ */}}
{{/* Common labels                                                */}}
{{/* ============================================================ */}}
{{- define "rinco.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{ include "rinco.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: rinco
{{- end }}

{{/* ============================================================ */}}
{{/* Selector labels                                              */}}
{{/* ============================================================ */}}
{{- define "rinco.selectorLabels" -}}
app.kubernetes.io/name: {{ include "rinco.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/* ============================================================ */}}
{{/* Component label                                              */}}
{{/* ============================================================ */}}
{{- define "rinco.componentLabel" -}}
app.kubernetes.io/component: {{ .component }}
app.kubernetes.io/name: {{ include "rinco.name" . }}-{{ .component }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/* ============================================================ */}}
{{/* Chart name                                                   */}}
{{/* ============================================================ */}}
{{- define "rinco.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/* ============================================================ */}}
{{/* Fullname                                                     */}}
{{/* ============================================================ */}}
{{- define "rinco.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end }}

{{/* ============================================================ */}}
{{/* Service fullname:  <release>-<chart>-<component>             */}}
{{/* ============================================================ */}}
{{- define "rinco.serviceName" -}}
{{- $name := include "rinco.fullname" . -}}
{{- $name = printf "%s-%s" $name .component -}}
{{- $name | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/* ============================================================ */}}
{{/* PostgreSQL host                                              */}}
{{/* ============================================================ */}}
{{- define "rinco.postgresql.host" -}}
{{- printf "%s-postgresql" (include "rinco.fullname" .) -}}
{{- end }}

{{/* ============================================================ */}}
{{/* Scylla host                                                  */}}
{{/* ============================================================ */}}
{{- define "rinco.scylla.host" -}}
{{- printf "%s-scylla" (include "rinco.fullname" .) -}}
{{- end }}

{{/* ============================================================ */}}
{{/* ClickHouse host                                              */}}
{{/* ============================================================ */}}
{{- define "rinco.clickhouse.host" -}}
{{- printf "%s-clickhouse" (include "rinco.fullname" .) -}}
{{- end }}

{{/* ============================================================ */}}
{{/* Image                                                       */}}
{{/* ============================================================ */}}
{{- define "rinco.image" -}}
{{- printf "%s/%s" .Values.image.registry .component -}}
{{- end }}

{{/* ============================================================ */}}
{{/* Image ref (with optional tag)                               */}}
{{/* ============================================================ */}}
{{- define "rinco.imageref" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" (include "rinco.image" (dict "Values" .Values "component" .component)) $tag -}}
{{- end }}

{{/* ============================================================ */}}
{{/* Common env                                                  */}}
{{/* ============================================================ */}}
{{- define "rinco.envCommon" -}}
- name: ENV
  value: {{ .Values.env }}
- name: LOG_LEVEL
  value: {{ .Values.logLevel }}
- name: TZ
  value: Asia/Ho_Chi_Minh
- name: VERSION
  value: {{ .Chart.AppVersion }}
- name: OTEL_EXPORTER_OTLP_ENDPOINT
  value: http://{{ include "rinco.fullname" . }}-otel-collector:4317
- name: NATS_URL
  value: nats://{{ include "rinco.fullname" . }}-nats:4222
- name: VALKEY_URL
  value: redis://{{ include "rinco.fullname" . }}-valkey-master:6379
{{- end }}
