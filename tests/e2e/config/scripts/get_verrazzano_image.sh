#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
WORKSPACE_ROOT=${SCRIPT_DIR}/../../../..
IMG_LIST_FILE=$1

if [ -z "${KUBECONFIG}" ] ; then
    echo "KUBECONFIG env var must be set!"
    exit 1
fi

# get image list from cluster and persist to output file
podlist=$(kubectl get pods -n $1 -o jsonpath="{..image}" |\tr -s '[[:space:]]' '\n' |\sort |\uniq | grep verrazzano | grep / | cut -d/ -f2- | grep -v fluentd)

printf '%s\n' "${podlist[@]}"
