#!/bin/bash
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

declare _files="$*"

cat <<EOF
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: api-nginx-conf
  namespace: {{ .Release.Namespace }}
  labels:
    app: {{ .Values.api.name }}
data:
{{- with .Values.api.proxy }}
EOF

for _i in ${_files}
do
    _filename=$(grep "Filename = \"" ${_i} | sed -e 's;.*\"\([^\"]*\)\".*$;\1;')
    echo "  ${_filename}: |"
    case "${_filename}" in
    startup.sh|reload.sh|nginx.conf)    _indent='s/^/    /' ;;
    *)                                  _indent='' ;;
    esac
    awk '{
        idx = index($0, "`")

        if (idx > 1)
            doprint = 1

        if (idx == 1)
            doprint = 0

        if (idx == 0 && doprint > 0)
            print
    }' ${_i} | sed -e "${_indent}"
done

echo '{{- end }}'
