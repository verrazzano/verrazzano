#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
PROMETHEUS_GW_URL=$1
PROMETHEUS_CRED=$2
JOB=$3
INS=$4
URL="${PROMETHEUS_GW_URL}/metrics/job/$JOB/instance/$INS"
if [ -z "$4" ]
  then
    URL="${PROMETHEUS_GW_URL}/metrics/job/$JOB"
fi
# curl -i -X DELETE "$URL" -u $PROMETHEUS_CRED
