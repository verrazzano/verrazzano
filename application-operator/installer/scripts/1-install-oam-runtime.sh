#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. $SCRIPT_DIR/common.sh

NAMESPACE="verrazzano-system"

function install_oam {
  log "Create ${NAMESPACE} namespace"
  if ! kubectl get namespace "${NAMESPACE}" > /dev/null 2>&1 ; then
    kubectl create namespace "${NAMESPACE}"
    if [ $? -ne 0 ]; then
      error "Failed to create ${NAMESPACE} namespace."
      return 1
    fi
  fi

  log "Add OAM Helm repository"
  helm repo add crossplane-master https://charts.crossplane.io/master/
  if [ $? -ne 0 ]; then
    error "Failed add OAM Helm repository."
    return 1
  fi

  log "Install OAM"
  helm install oam \
      --namespace "${NAMESPACE}" \
      --set image.repostiory=ghcr.io/verrazzano/oam-kubernetes-runtime \
      --set image.tag=v0.3.0 \
      --version 0.3.0 \
      crossplane-master/oam-kubernetes-runtime
  if [ $? -ne 0 ]; then
    error "Failed to OAM Helm install."
    return 1
  fi

  log "Setup OAM roles"
  kubectl create clusterrolebinding cluster-admin-binding-oam --clusterrole cluster-admin --user system:serviceaccount:verrazzano-system:oam-kubernetes-runtime-oam
  if [ $? -ne 0 ]; then
    error "Failed to setup OAM roles."
    return 1
  fi

  echo "Wait for OAM runtime pod to be ready."
  attempt=1
  while true; do
    kubectl -n verrazzano-system wait --for=condition=ready pods --selector='app.kubernetes.io/name=oam-kubernetes-runtime' --timeout 15s
    if [ $? -eq 0 ]; then
      echo "OAM runtime pods found ready on attempt ${attempt}."
      break
    elif [ ${attempt} -eq 1 ]; then
      echo "No OAM runtime pods found ready on initial attempt. Retrying after delay."
    elif [ ${attempt} -ge 20 ]; then
      echo "ERROR: No OAM runtime pods found ready after ${attempt} attempts. Listing pods."
      kubectl get pods -n verrazzano-system
      echo "ERROR: Exiting."
      return 1
    fi
    attempt=$(($attempt+1))
    sleep .5
  done

}

action "Installing OAM runtime" install_oam || fail "Failed to install OAM runtime."
