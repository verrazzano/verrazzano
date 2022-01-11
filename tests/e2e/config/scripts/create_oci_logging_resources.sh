#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function usage() {
    echo """
Creates OCI Logging Log Group and Log resources.

Usage:

    $0 <compartment_id>
"""
exit 1
}

if [[ -z "$1" || "$1" == "-h" ]]; then
    usage
fi

COMPARTMENT_ID=$1

# Create the OCI Log Group that will contain the OCI Log objects

SUFFIX=$(uuidgen | cut -d'-' -f1)

LOG_GROUP_RESPONSE=$(oci logging log-group create --compartment-id ${COMPARTMENT_ID} --display-name "log-group-${SUFFIX}" --description "Created by test automation" --wait-for-state SUCCEEDED)
if [ $? -ne 0 ]; then
    echo Failed creating OCI Log Group
    exit 1
fi

LOG_GROUP_ID=$(echo ${LOG_GROUP_RESPONSE} | jq -r '.data.resources[].identifier')

# Create an OCI Log for system logs
LOG_RESPONSE=$(oci logging log create --log-group-id ${LOG_GROUP_ID} --display-name "system-log-${SUFFIX}" --log-type CUSTOM --wait-for-state SUCCEEDED)
if [ $? -ne 0 ]; then
    echo Failed creating OCI Log for system logs
    exit 1
fi

SYSTEM_LOG_ID=$(echo ${LOG_RESPONSE} | jq -r '.data.resources[].identifier')

# Create an OCI Log for app logs
LOG_RESPONSE=$(oci logging log create --log-group-id ${LOG_GROUP_ID} --display-name "app-log-${SUFFIX}" --log-type CUSTOM --wait-for-state SUCCEEDED)
if [ $? -ne 0 ]; then
    echo Failed creating OCI Log for app logs
    exit 1
fi

APP_LOG_ID=$(echo ${LOG_RESPONSE} | jq -r '.data.resources[].identifier')

# Output results in json
echo "{\"logGroupId\":\"${LOG_GROUP_ID}\",\"systemLogId\":\"${SYSTEM_LOG_ID}\",\"appLogId\":\"${APP_LOG_ID}\",\"suffix\":\"${SUFFIX}\"}"
