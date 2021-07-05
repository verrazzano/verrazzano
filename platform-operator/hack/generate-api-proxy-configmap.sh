#!/bin/bash
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

#
# Shell script to generate the verrazzano-api-proxy-configmap.yaml file from go templates
# that are the SoT for the proxy code. Template values are not substituted because that
# happens during the helm install.
#

declare _files="$*"

#
# Generate and output the configmap metadata
#

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

#
# For each input template file (provided on the command line), do:
#
# 1. Get the filename by pulling the "Filename" constant out of the template file
#    and stripping away the golang code surrounding the actual constant value.
#
# 2. Run the entire file through an awk script that strips away everything before
#    and after the string constant that contains the actual template.
#
# Note that this code depends on there being exactly two constants in each template
# file, and on canonical naming for those constants, and also depends the quote
# characters used, and their placement.
#

for _i in ${_files}
do
    # Get the filename constant from the file, and strip away the golang syntax surrounding
    # the constant value; i.e., everything except what's between the two double quotes.

    _filename=$(grep "Filename = \"" ${_i} | sed -e 's;.*\"\([^\"]*\)\".*$;\1;')

    # Output the filename
    echo "  ${_filename}: |"

    # Run the entire file through awk to strip out the golang code before and after
    # the actual value of the constant that defines the template itself.
    #
    # The template value is quoted using backtick quotes (`). We look for that character
    # on each line. If we find a backtick that is not the first character on the line,
    # that indicates the start of the constant value. We print the part of line that
    # comes after the quote, and also start printing the lines that follow. If we find
    # a backtick as the first character on a line, that indicates the end of the constant,
    # and we stop printing lines.

    awk '{
        idx = index($0, "`")

        if (idx > 1) {
            doprint = 1
            split($0, items, "`")
            printf("    %s\n", items[2])
        }

        if (idx == 1)
            doprint = 0

        if (idx == 0 && doprint > 0)
            print
    }' ${_i}
done

#
# Terminate the "{{- with .Values.api.proxy }}" that we started above
#

echo '{{ end }}'

