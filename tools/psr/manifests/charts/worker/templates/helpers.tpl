# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Note: This file is normally named _helpers.tpl, but the pscctl command line tool embeds this chart
# using golang embed, however if the file has an "_" prefix, then it is excluded from the go binary

{{- define "worker.fullName" -}}
{{- print .Release.Name "-" .Values.global.envVars.PSR_WORKER_TYPE -}}
{{- end -}}


