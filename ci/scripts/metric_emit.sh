#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
PROMETHEUS_GW_URL=$1
SAURON_CRED=$2
JOB=$3
BRANCH=$4
# BRANCH is used as "instance" for cleanup
INSTANCE=$(echo $4 | sed -e "s/\//_/g")
LABELS=$(echo $5 | tr -d \')
LABELS="${LABELS},job=\"${JOB}\",branch=\"${BRANCH}\""
STATUS=$6
DURATION=$7
TIMESTAMP=$(date +%s)
TIME_METRIC=""
if [ $DURATION -gt 0 ]
then
    TIME_METRIC="${JOB}_time{${LABELS}} $DURATION $TIMESTAMP"
fi
echo "Sending to ${PROMETHEUS_GW_URL}metrics/job/${JOB}"
cat <<EOF | curl -i --data-binary @- ${PROMETHEUS_GW_URL}metrics/job/${JOB}/instance/${INSTANCE} -u $SAURON_CRED
${JOB}_status{${LABELS}} $STATUS $TIMESTAMP
${TIME_METRIC}
EOF
