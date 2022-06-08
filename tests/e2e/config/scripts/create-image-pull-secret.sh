#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -u

NAME=$1
DOCKER_SERVER=$2
USERNAME=$3
PASSWORD=$4
NAMESPACE=${5:-default}

if [ -z "${KUBECONFIG}" ] ; then
    echo "KUBECONFIG env var must be set!"
    exit 1
fi

if kubectl get secret -n ${NAMESPACE} ${NAME} 2>&1 > /dev/null; then
  echo "Secret ${NAME} already exists"
  exit 0
fi

set +x # always disable shell debug for this
kubectl create secret docker-registry ${NAME} \
                            --docker-server=${DOCKER_SERVER} \
                            --docker-username=${USERNAME} \
                            --docker-password=${PASSWORD} \
                            -n ${NAMESPACE}
