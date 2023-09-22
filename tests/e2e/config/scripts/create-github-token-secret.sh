#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -u

NAME=$1
GITHUB_TOKEN=$2
NAMESPACE=${3:-"verrazzano-install"}

if [ -z "${KUBECONFIG}" ] ; then
    echo "KUBECONFIG env var must be set!"
    exit 1
fi

if kubectl get secret -n ${NAMESPACE} ${NAME} 2>&1 > /dev/null; then
  echo "Secret ${NAME} already exists"
  exit 0
fi

set +x # always disable shell debug for this
kubectl create secret generic ${NAME} \
                            --from-literal=GITHUB_TOKEN="${GITHUB_TOKEN}" \
                            -n ${NAMESPACE}
