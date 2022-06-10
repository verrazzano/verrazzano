#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
CLUSTER_NAME=$1
KUBECONFIG=$2

if [ -z "${KUBECONFIG}" ]; then
  echo "KUBECONFIG must be set"
  exit 1
fi

echo "Delete the cluster and kube config in multi-cluster environment"
kind delete cluster --name ${CLUSTER_NAME}
if [ -f "${KUBECONFIG}" ]
then
  echo "Deleting the kubeconfig '${KUBECONFIG}' ..."
  rm ${KUBECONFIG}
fi

