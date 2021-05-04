#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# This script waits for the KIND control plane node to reach Ready status, with a default wait period of 300 seconds.
# The script assumes the cluster access configuration is stored in ${HOME}/.kube/config or environment variable
# KUBECONFIG is set appropriately.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
CLUSTER_NAME=${1:-"kind"}
MAX_WAIT=${2:-300}
POLLING_INTERVAL=${3:-10}
LIST_PODS=${4:-true}

isNodeReady=false

for ((loopCount=0; loopCount<$MAX_WAIT; loopCount+=$POLLING_INTERVAL)); do
  if [[ "$(kubectl get node "${CLUSTER_NAME}"-control-plane -o 'jsonpath={.status.conditions[?(@.type=="Ready")].status}')" == 'True' ]]; then
    echo "The node "${CLUSTER_NAME}"-control-plane is ready."
    isNodeReady=true
    break
  fi
  echo "Wait for "$POLLING_INTERVAL seconds before checking the status of "${CLUSTER_NAME}"-control-plane" ..."
  sleep $POLLING_INTERVAL
done

if [ "$LIST_PODS" = true ] ; then
  echo "Listing pods in kube-system namespace ..."
  kubectl get pods -n kube-system
fi

if [[ $isNodeReady != true ]]; then
  echo "The node "${CLUSTER_NAME}"-control-plane is not ready after $MAX_WAIT seconds, exiting ..."
  exit 1
fi
