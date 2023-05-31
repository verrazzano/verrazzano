# Portions of the code in this file are derived from https://github.com/cert-manager/webhook-example/blob/master/deploy/example-webhook/templates/_helpers.tpl
# Portions of the code in this file are derived from https://gitlab.com/dn13/cert-manager-webhook-oci/-/blob/1.1.0/deploy/cert-manager-webhook-oci/templates/_helpers.tpl

{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "cert-manager-webhook-oci.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cert-manager-webhook-oci.fullname" -}}
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
{{- end -}}

{{/*
Default the Cert-Manager clusterResourceNamespace to the Cert-Manager namespace if not explicitly set
*/}}
{{- define "cert-manager-webhook-oci.clusterResourceNamespace" -}}
{{- if .Values.certManager.clusterResourceNamespace -}}
{{- printf "%s" .Values.certManager.clusterResourceNamespace -}}
{{- else -}}
{{- printf "%s" .Values.certManager.namespace -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cert-manager-webhook-oci.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cert-manager-webhook-oci.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "cert-manager-webhook-oci.fullname" .) }}
{{- end -}}

{{- define "cert-manager-webhook-oci.rootCAIssuer" -}}
{{ printf "%s-ca" (include "cert-manager-webhook-oci.fullname" .) }}
{{- end -}}

{{- define "cert-manager-webhook-oci.rootCACertificate" -}}
{{ printf "%s-ca" (include "cert-manager-webhook-oci.fullname" .) }}
{{- end -}}

{{- define "cert-manager-webhook-oci.servingCertificate" -}}
{{ printf "%s-webhook-tls" (include "cert-manager-webhook-oci.fullname" .) }}
{{- end -}}

