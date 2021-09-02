#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "$SAURON_CRED" ] || [ -z "$BRANCH_NAME" ] || [ -z "$PROMETHEUS_GW_URL" ] ; then
  echo "The script expects environment variables PROMETHEUS_GW_UR, SAURON_CRED and BRANCH_NAME."
  exit 1
fi

JOB=$1
# BRANCH_NAME is used as "instance" for cleanup
INSTANCE=$(echo "$BRANCH_NAME" | sed -e "s/\//_/g")
LABELS=$(echo $2 | tr -d \')
LABELS="${LABELS},job=\"${JOB}\",branch=\"${BRANCH_NAME}\""
STATUS=$3
DURATION=$4
TIMESTAMP=$(date +%s)
TIME_METRIC=""
if [ $DURATION -gt 0 ]
then
    TIME_METRIC="${JOB}_time{${LABELS}} $DURATION $TIMESTAMP"
fi
echo "Pushing metric to ${PROMETHEUS_GW_URL}metrics/job/${JOB}"
cat <<EOF | curl -i --data-binary @- ${PROMETHEUS_GW_URL}metrics/job/${JOB}/instance/${INSTANCE} -u $SAURON_CRED
${JOB}_status{${LABELS}} $STATUS $TIMESTAMP
${TIME_METRIC}
EOF
