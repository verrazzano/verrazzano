#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
WORK_DIR=$(cd $(dirname "$0"); pwd -P)
TMP_DIR=$(mktemp -d)
OC_TELEMETRY_URL=$1
COMPARTMENT=$2
NAMESPACE=$3
JOB=$4
# BRANCH=$(echo-client $5 | sed -e "s/\//\\\\\//g")
BRANCH=$5
# LABELS=$5
LABELS=$(echo $6 | tr -d \')
LABELS="${LABELS},\"job\":\"${JOB}\",\"branch\":\"${BRANCH}\""
LABELS=$(echo "${LABELS}" | tr '=' ':')
STATUS=$7
DURATION=$8
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

cat <<EOF>>${TMP_DIR}/oci_metric_data.json
{
    "metricData": [
        {
            "namespace": "${NAMESPACE}",
            "name": "${JOB}.status",
            "compartmentId": "${COMPARTMENT}",
            "dimensions": {
                ${LABELS}
            },
            "metadata": {
                "unit": "boolean"
            },
            "datapoints": [
                {
                    "timestamp": "${TIMESTAMP}",
                    "value": ${STATUS}
                }
            ]
        },
        {
            "namespace": "${NAMESPACE}",
            "name": "${JOB}.duration",
            "compartmentId": "${COMPARTMENT}",
            "dimensions": {
                ${LABELS}
            },
            "metadata": {
                "unit": "second"
            },
            "datapoints": [
                {
                    "timestamp": "${TIMESTAMP}",
                    "value": ${DURATION}
                }
            ]
        }
    ]
}
EOF
# cat ${TMP_DIR}/oci_metric_data.json
oci monitoring metric-data post --from-json file://${TMP_DIR}/oci_metric_data.json --endpoint "${OC_TELEMETRY_URL}"
