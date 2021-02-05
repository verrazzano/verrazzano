#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

NAME=$1
DOCKER_SERVER=$2
USERNAME=$3
PASSWORD=$4
NAMESPACE=${5:-default}

if [ -z "${KUBECONFIG}" ] ; then
    echo "KUBECONFIG env var must be set!"
    exit 1
fi

kubectl create secret docker-registry ${NAME} \
                            --docker-server=${DOCKER_SERVER} \
                            --docker-username=${USERNAME} \
                            --docker-password=${PASSWORD} \
                            -n ${NAMESPACE}
