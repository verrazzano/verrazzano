#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -e

NAME=$1
DOCKER_SERVER=$2
USERNAME=$3
PASSWORD=$4

if [ -z "${KUBECONFIG}" ] ; then
    echo "KUBECONFIG env var must be set!"
    exit 1
fi

# Create the verrazzano-system namespace if it does not exist already
if ! kubectl get namespace verrazzano-system > /dev/null 2>&1 ; then
  kubectl create  ns verrazzano-system
fi

# Create the docker-registry secret ${NAME} if it does not exist already
if ! kubectl get secret -n verrazzano-system ${NAME} > /dev/null 2>&1 ; then
  kubectl create secret -n verrazzano-system docker-registry ${NAME} \
                            --docker-server=${DOCKER_SERVER} \
                            --docker-username=${USERNAME} \
                            --docker-password=${PASSWORD}
fi
