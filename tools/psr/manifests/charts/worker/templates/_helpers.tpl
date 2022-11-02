# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

{{- define "worker.fullName" -}}
{{- print .Release.Name "-" .Values.global.envVars.PSR_WORKER_TYPE -}}
{{- end -}}


