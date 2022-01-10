#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function usage() {
    echo """
Deletes OCI Logging Log Group and Log resources.

Usage:

    $0 <log_group_id> <system_log_id> <app_log_id>
"""
exit 1
}

if [[ -z "$1" || "$1" == "-h" || "$#" -ne 3 ]]; then
    usage
fi

LOG_GROUP_ID=$1
SYSTEM_LOG_ID=$2
APP_LOG_ID=$3

# Make a best-effort to delete all of the resources (don't exit on failure)

# Log objects must be deleted before the Log Group
oci logging log delete --log-group-id ${LOG_GROUP_ID} --log-id ${SYSTEM_LOG_ID} --force --wait-for-state SUCCEEDED
if [ $? -ne 0 ]; then
    echo Failed deleting OCI Log for system logs
fi

oci logging log delete --log-group-id ${LOG_GROUP_ID} --log-id ${APP_LOG_ID} --force --wait-for-state SUCCEEDED
if [ $? -ne 0 ]; then
    echo Failed deleting OCI Log for app logs
fi

# Delete the Log Group
oci logging log-group delete --log-group-id ${LOG_GROUP_ID} --force --wait-for-state SUCCEEDED
if [ $? -ne 0 ]; then
    echo Failed deleting OCI Log Group
fi
